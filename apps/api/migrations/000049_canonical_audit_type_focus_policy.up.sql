-- The approved AGA source is shared across audit types. This bounded,
-- server-owned policy turns the selected audit type into an advisory focus
-- signal without removing any immutable question from the manager's catalog.
-- A manager can always clear the recommendation/focus filters and select an
-- otherwise valid question manually.
CREATE FUNCTION canonical_audit_type_matches_question_focus(
    audit_type text,
    inspection_type_codes text[]
) RETURNS boolean
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT CASE
        WHEN audit_type IS NULL OR btrim(audit_type) = '' THEN true
        WHEN audit_type = 'RAMP_INSPECTION' THEN
            COALESCE(inspection_type_codes, ARRAY[]::text[])
                && ARRAY['ON_SITE_INSPECTION', 'PERIODIC_SURVEILLANCE']::text[]
        WHEN audit_type = 'CABIN_INSPECTION' THEN
            COALESCE(inspection_type_codes, ARRAY[]::text[])
                && ARRAY['DOCUMENT_AND_RECORD_REVIEW', 'PERIODIC_SURVEILLANCE']::text[]
        ELSE true
    END
$$;
