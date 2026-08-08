#!/bin/sh
set -eu

read_secret() {
  secret_path=$1
  if [ ! -f "$secret_path" ]; then
    echo "required mounted local-preprod object-store credential is unavailable" >&2
    exit 1
  fi
  secret_value=$(tr -d '\r\n' <"$secret_path")
  if [ -z "$secret_value" ]; then
    echo "required mounted local-preprod object-store credential is empty" >&2
    exit 1
  fi
  printf '%s' "$secret_value"
}

run_sensitive_mc() {
  if ! "$@" >/dev/null 2>&1; then
    echo "local-preprod object-store credential administration failed" >&2
    return 1
  fi
}

bucket=${AVIA_PREPROD_OBJECT_BUCKET:-}
if [ "$bucket" != "aviasurveil360-local-preprod" ]; then
  echo "local-preprod object bucket must be exactly aviasurveil360-local-preprod" >&2
  exit 1
fi

root_user=$(read_secret "${MINIO_ROOT_USER_FILE:-/run/secrets/preprod_minio_root_user}")
root_password=$(read_secret "${MINIO_ROOT_PASSWORD_FILE:-/run/secrets/preprod_minio_root_password}")
loader_access_key=$(read_secret "${MINIO_LOADER_ACCESS_KEY_FILE:-/run/secrets/preprod_minio_api_access_key}")
loader_secret_key=$(read_secret "${MINIO_LOADER_SECRET_KEY_FILE:-/run/secrets/preprod_minio_api_secret_key}")

export MINIO_ROOT_USER="$root_user"
export MINIO_ROOT_PASSWORD="$root_password"

minio server /data --console-address :9001 &
server_pid=$!

stop_server() {
  kill "$server_pid" 2>/dev/null || true
  wait "$server_pid" 2>/dev/null || true
}
trap stop_server EXIT HUP INT TERM

alias=local-preprod
until mc alias set "$alias" http://127.0.0.1:9000 "$root_user" "$root_password" >/dev/null 2>&1; do
  if ! kill -0 "$server_pid" 2>/dev/null; then
    echo "local-preprod MinIO stopped before private namespace initialization" >&2
    exit 1
  fi
  sleep 1
done

for private_bucket in \
  "$bucket" \
  evidence-quarantine \
  evidence-clean \
  inspection-attachments \
  generated-documents; do
  mc mb --ignore-existing "$alias/$private_bucket"
  mc anonymous set none "$alias/$private_bucket"
  mc version enable "$alias/$private_bucket"
done

cat >/tmp/preprod-loader-policy.json <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["s3:GetBucketLocation", "s3:ListBucket"],
      "Resource": [
        "arn:aws:s3:::aviasurveil360-local-preprod",
        "arn:aws:s3:::evidence-quarantine",
        "arn:aws:s3:::evidence-clean",
        "arn:aws:s3:::inspection-attachments",
        "arn:aws:s3:::generated-documents"
      ]
    },
    {
      "Effect": "Allow",
      "Action": ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"],
      "Resource": [
        "arn:aws:s3:::aviasurveil360-local-preprod/runs/*",
        "arn:aws:s3:::evidence-quarantine/*",
        "arn:aws:s3:::evidence-clean/*",
        "arn:aws:s3:::inspection-attachments/*",
        "arn:aws:s3:::generated-documents/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": ["s3:ListBucket"],
      "Resource": ["arn:aws:s3:::aviasurveil360-local-preprod"],
      "Condition": {
        "StringLike": {
          "s3:prefix": ["runs/*"]
        }
      }
    }
  ]
}
EOF

run_sensitive_mc mc admin user add "$alias" "$loader_access_key" "$loader_secret_key"
mc admin policy create "$alias" aviasurveil360-local-preprod-loader /tmp/preprod-loader-policy.json
run_sensitive_mc mc admin policy attach \
  "$alias" aviasurveil360-local-preprod-loader \
  --user "$loader_access_key"

rm -f /tmp/preprod-loader-policy.json
touch /tmp/preprod-minio-initialized
unset root_user root_password loader_access_key loader_secret_key

wait "$server_pid"
