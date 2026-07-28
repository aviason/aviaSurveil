#!/bin/sh
set -eu

read_secret() {
  value=$(tr -d '\r\n' <"$1")
  if [ -z "$value" ]; then
    echo "required backup-store credential is empty" >&2
    exit 1
  fi
  printf '%s' "$value"
}

run_sensitive_mc() {
  if ! "$@" >/dev/null 2>&1; then
    echo "backup-store credential administration failed" >&2
    return 1
  fi
}

root_user=$(read_secret /run/secrets/backup_minio_root_user)
root_password=$(read_secret /run/secrets/backup_minio_root_password)
pgbackrest_user=$(read_secret /run/secrets/backup_pgbackrest_access_key)
pgbackrest_password=$(read_secret /run/secrets/backup_pgbackrest_secret_key)
object_user=$(read_secret /run/secrets/backup_object_access_key)
object_password=$(read_secret /run/secrets/backup_object_secret_key)

export MINIO_ROOT_USER="$root_user"
export MINIO_ROOT_PASSWORD="$root_password"

minio server /data --certs-dir /certs --console-address :9001 &
server_pid=$!

stop_server() {
  kill "$server_pid" 2>/dev/null || true
  wait "$server_pid" 2>/dev/null || true
}
trap stop_server EXIT HUP INT TERM

alias=recovery
until mc alias set --insecure "$alias" https://127.0.0.1:9000 \
  "$root_user" "$root_password" >/dev/null 2>&1; do
  if ! kill -0 "$server_pid" 2>/dev/null; then
    echo "backup MinIO stopped before initialization" >&2
    exit 1
  fi
  sleep 1
done

create_locked_bucket() {
  bucket=$1
  mc mb --insecure --ignore-existing --with-lock "$alias/$bucket"
  mc anonymous set --insecure none "$alias/$bucket"
  mc version enable --insecure "$alias/$bucket"
}

create_locked_bucket application-database-backups
create_locked_bucket identity-database-backups
create_locked_bucket application-object-backups
create_locked_bucket recovery-catalog

cat >/tmp/pgbackrest-policy.json <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["s3:GetBucketLocation", "s3:ListBucket"],
      "Resource": [
        "arn:aws:s3:::application-database-backups",
        "arn:aws:s3:::identity-database-backups"
      ]
    },
    {
      "Effect": "Allow",
      "Action": ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"],
      "Resource": [
        "arn:aws:s3:::application-database-backups/*",
        "arn:aws:s3:::identity-database-backups/*"
      ]
    }
  ]
}
EOF

cat >/tmp/object-policy.json <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["s3:GetBucketLocation", "s3:ListBucket"],
      "Resource": [
        "arn:aws:s3:::application-object-backups",
        "arn:aws:s3:::recovery-catalog"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:PutObjectRetention"
      ],
      "Resource": [
        "arn:aws:s3:::application-object-backups/*",
        "arn:aws:s3:::recovery-catalog/*"
      ]
    }
  ]
}
EOF

run_sensitive_mc mc admin --insecure user add "$alias" \
  "$pgbackrest_user" "$pgbackrest_password"
mc admin --insecure policy create "$alias" avia-pgbackrest /tmp/pgbackrest-policy.json
run_sensitive_mc mc admin --insecure policy attach "$alias" avia-pgbackrest \
  --user "$pgbackrest_user"

run_sensitive_mc mc admin --insecure user add "$alias" "$object_user" "$object_password"
mc admin --insecure policy create "$alias" avia-object-backup /tmp/object-policy.json
run_sensitive_mc mc admin --insecure policy attach "$alias" avia-object-backup \
  --user "$object_user"

rm -f /tmp/pgbackrest-policy.json /tmp/object-policy.json
touch /tmp/backup-minio-initialized
unset root_user root_password pgbackrest_user pgbackrest_password
unset object_user object_password

wait "$server_pid"
