-- An Inspection Attachment registration is mutable workflow state, while a
-- completed object is an immutable evidentiary artifact.  Pin every complete
-- upload to a distinct append-only version so later scan/projection updates
-- cannot overwrite the object identity that was originally received.
CREATE TABLE inspection_attachment_versions (
    id text PRIMARY KEY,
    inspection_attachment_id text NOT NULL REFERENCES inspection_attachments(id),
    version bigint NOT NULL CHECK (version > 0),
    organization_id text NOT NULL REFERENCES organizations(id),
    source_object_metadata_id text NOT NULL REFERENCES object_metadata(id),
    upload_session_id text REFERENCES upload_sessions(id),
    file_name text NOT NULL,
    media_type text NOT NULL,
    sha256 text NOT NULL,
    size_bytes bigint NOT NULL CHECK (size_bytes >= 0),
    submitted_by_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    submitted_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (inspection_attachment_id, version),
    UNIQUE (id, inspection_attachment_id),
    UNIQUE (id, source_object_metadata_id),
    UNIQUE (upload_session_id)
);

CREATE INDEX inspection_attachment_versions_attachment_idx
    ON inspection_attachment_versions (inspection_attachment_id, version DESC);

CREATE TRIGGER inspection_attachment_versions_append_only
BEFORE UPDATE OR DELETE ON inspection_attachment_versions
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();

-- Keep the immutable receipt in the exact tenant that owns its attachment.
ALTER TABLE inspection_attachments
    ADD CONSTRAINT inspection_attachments_id_organization_key
    UNIQUE (id, organization_id);

ALTER TABLE inspection_attachment_versions
    ADD CONSTRAINT inspection_attachment_versions_attachment_organization_fkey
    FOREIGN KEY (inspection_attachment_id, organization_id)
    REFERENCES inspection_attachments (id, organization_id);

ALTER TABLE inspection_attachments
    ADD COLUMN current_version_id text;

-- Preserve historical uploads that already reached an object metadata record.
-- Attachments without a completed object deliberately remain without a
-- version and fail closed until a normal Complete command creates one.
INSERT INTO inspection_attachment_versions (
    id, inspection_attachment_id, version, organization_id,
    source_object_metadata_id, upload_session_id, file_name, media_type,
    sha256, size_bytes, submitted_by_subject_id, submitted_at, created_at
)
SELECT 'inspection-attachment-version:legacy:' || attachment.id,
       attachment.id,
       1,
       attachment.organization_id,
       attachment.object_metadata_id,
       metadata.upload_id,
       attachment.file_name,
       attachment.declared_media_type,
       attachment.declared_sha256,
       attachment.declared_size_bytes,
       attachment.created_by_subject_id,
       COALESCE(session.completed_at, attachment.updated_at),
       attachment.updated_at
FROM inspection_attachments attachment
JOIN object_metadata metadata ON metadata.id = attachment.object_metadata_id
LEFT JOIN upload_sessions session ON session.id = metadata.upload_id
WHERE attachment.object_metadata_id IS NOT NULL
ON CONFLICT (inspection_attachment_id, version) DO NOTHING;

UPDATE inspection_attachments attachment
SET current_version_id = version.id
FROM inspection_attachment_versions version
WHERE version.inspection_attachment_id = attachment.id
  AND attachment.current_version_id IS NULL;

ALTER TABLE inspection_attachments
    ADD CONSTRAINT inspection_attachments_current_version_fkey
    FOREIGN KEY (current_version_id, id)
    REFERENCES inspection_attachment_versions (id, inspection_attachment_id);

-- A mutable attachment may only point at the source object recorded by its
-- current immutable version. This prevents later workflow updates from
-- decoupling a version identity from the object that was actually received.
ALTER TABLE inspection_attachments
    ADD CONSTRAINT inspection_attachments_current_version_source_fkey
    FOREIGN KEY (current_version_id, object_metadata_id)
    REFERENCES inspection_attachment_versions (id, source_object_metadata_id);

ALTER TABLE inspection_attachments
    ADD CONSTRAINT inspection_attachments_current_version_object_pair_check
    CHECK ((current_version_id IS NULL) = (object_metadata_id IS NULL));
