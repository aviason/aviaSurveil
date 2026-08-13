ALTER TABLE user_lifecycle_requests
    ADD COLUMN requested_email text,
    ADD COLUMN requested_display_name text;

ALTER TABLE user_lifecycle_requests
    ADD CONSTRAINT user_lifecycle_identity_shape_check
    CHECK (
        (
            requested_action = 'PROVISION'
            AND requested_email IS NOT NULL
            AND requested_display_name IS NOT NULL
            AND (
                (status = 'SUCCEEDED' AND subject_id IS NOT NULL)
                OR
                (status <> 'SUCCEEDED' AND subject_id IS NULL)
            )
        )
        OR
        (
            requested_action <> 'PROVISION'
            AND subject_id IS NOT NULL
            AND requested_email IS NULL
            AND requested_display_name IS NULL
        )
    ) NOT VALID;

CREATE UNIQUE INDEX user_lifecycle_active_provision_email_idx
    ON user_lifecycle_requests (lower(requested_email))
    WHERE requested_action = 'PROVISION'
      AND status IN ('PENDING', 'RUNNING', 'SUCCEEDED');
