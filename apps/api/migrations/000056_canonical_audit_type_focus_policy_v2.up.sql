-- Keep the selected audit-type focus deterministic for every supported
-- canonical execution type. This replaces the earlier two-type policy without
-- mutating any catalog, question, or historical Audit row.
CREATE OR REPLACE FUNCTION canonical_audit_type_matches_question_focus(
    audit_type text,
    inspection_type_codes text[]
) RETURNS boolean
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT CASE upper(btrim(COALESCE(audit_type, '')))
        WHEN '' THEN true
        WHEN 'RAMP' THEN COALESCE(inspection_type_codes, ARRAY[]::text[]) && ARRAY['ON_SITE_INSPECTION', 'PERIODIC_SURVEILLANCE']::text[]
        WHEN 'RAMP_INSPECTION' THEN COALESCE(inspection_type_codes, ARRAY[]::text[]) && ARRAY['ON_SITE_INSPECTION', 'PERIODIC_SURVEILLANCE']::text[]
        WHEN 'CABIN' THEN COALESCE(inspection_type_codes, ARRAY[]::text[]) && ARRAY['DOCUMENT_AND_RECORD_REVIEW', 'PERIODIC_SURVEILLANCE']::text[]
        WHEN 'CABIN_INSPECTION' THEN COALESCE(inspection_type_codes, ARRAY[]::text[]) && ARRAY['DOCUMENT_AND_RECORD_REVIEW', 'PERIODIC_SURVEILLANCE']::text[]
        WHEN 'CHANGE_APPROVAL' THEN COALESCE(inspection_type_codes, ARRAY[]::text[]) && ARRAY['CHANGE_APPROVAL']::text[]
        WHEN 'DOCUMENT_AND_RECORD_REVIEW' THEN COALESCE(inspection_type_codes, ARRAY[]::text[]) && ARRAY['DOCUMENT_AND_RECORD_REVIEW']::text[]
        WHEN 'FOLLOW_UP' THEN COALESCE(inspection_type_codes, ARRAY[]::text[]) && ARRAY['FOLLOW_UP']::text[]
        WHEN 'INITIAL_CERTIFICATION' THEN COALESCE(inspection_type_codes, ARRAY[]::text[]) && ARRAY['INITIAL_CERTIFICATION']::text[]
        WHEN 'ON_SITE_INSPECTION' THEN COALESCE(inspection_type_codes, ARRAY[]::text[]) && ARRAY['ON_SITE_INSPECTION']::text[]
        WHEN 'PERIODIC_SURVEILLANCE' THEN COALESCE(inspection_type_codes, ARRAY[]::text[]) && ARRAY['PERIODIC_SURVEILLANCE']::text[]
        WHEN 'RENEWAL' THEN COALESCE(inspection_type_codes, ARRAY[]::text[]) && ARRAY['RENEWAL']::text[]
        WHEN 'SPECIAL_PURPOSE' THEN COALESCE(inspection_type_codes, ARRAY[]::text[]) && ARRAY['SPECIAL_PURPOSE']::text[]
        ELSE false
    END
$$;
