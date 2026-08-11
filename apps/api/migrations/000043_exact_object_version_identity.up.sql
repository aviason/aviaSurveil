ALTER TABLE object_metadata
    ADD COLUMN IF NOT EXISTS object_version_id text,
    ADD COLUMN IF NOT EXISTS object_etag text;

ALTER TABLE object_metadata
    DROP CONSTRAINT IF EXISTS object_metadata_exact_version_identity_check;

ALTER TABLE object_metadata
    ADD CONSTRAINT object_metadata_exact_version_identity_check
    CHECK (
        (object_version_id IS NULL AND object_etag IS NULL)
        OR
        (
            object_version_id IS NOT NULL
            AND object_etag IS NOT NULL
            AND btrim(object_version_id) <> ''
            AND btrim(object_etag) <> ''
        )
    );

CREATE UNIQUE INDEX IF NOT EXISTS object_metadata_bucket_key_version_uidx
    ON object_metadata (bucket_name, object_key, object_version_id)
    WHERE bucket_name IS NOT NULL AND object_version_id IS NOT NULL;
