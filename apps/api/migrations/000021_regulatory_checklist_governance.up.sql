CREATE TABLE caa_departments (
    id text PRIMARY KEY, name text NOT NULL UNIQUE,
    status text NOT NULL CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE caa_organizational_units (
    id text PRIMARY KEY, department_id text NOT NULL REFERENCES caa_departments(id), name text NOT NULL,
    status text NOT NULL CHECK (status IN ('ACTIVE', 'INACTIVE')), created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (department_id, name)
);
CREATE TABLE caa_department_memberships (
    id text PRIMARY KEY, subject_id text NOT NULL REFERENCES identity_references(subject_id),
    department_id text NOT NULL REFERENCES caa_departments(id), organizational_unit_id text REFERENCES caa_organizational_units(id),
    membership_role text NOT NULL CHECK (membership_role = 'DEPARTMENT_MANAGER'), effective_from date NOT NULL, effective_to date,
    status text NOT NULL CHECK (status IN ('ACTIVE', 'REVOKED', 'EXPIRED')), created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (effective_to IS NULL OR effective_to > effective_from), UNIQUE (subject_id, department_id, organizational_unit_id, effective_from)
);
CREATE INDEX caa_department_memberships_effective_authority_idx ON caa_department_memberships (subject_id, department_id, effective_from, effective_to) WHERE status = 'ACTIVE';

CREATE TABLE service_provider_types (
    id text PRIMARY KEY, catalog_version text NOT NULL CHECK (catalog_version = '1.0.0'), label text NOT NULL UNIQUE,
    raw_oversight_topics text NOT NULL, raw_responsible_caa_unit text NOT NULL, target_kinds text[] NOT NULL,
    normalization_status text NOT NULL CHECK (normalization_status IN ('NORMALIZED', 'REVIEW_REQUIRED')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (cardinality(target_kinds) > 0), CHECK (target_kinds <@ ARRAY['ORGANIZATION','PERSON','FACILITY','DEVICE','SYSTEM','ASSET','LOCATION']::text[])
);
CREATE TABLE service_provider_topics (id text PRIMARY KEY, topic text NOT NULL UNIQUE, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE service_provider_topic_links (
    service_provider_type_id text NOT NULL REFERENCES service_provider_types(id), service_provider_topic_id text NOT NULL REFERENCES service_provider_topics(id),
    ordinal integer NOT NULL CHECK (ordinal > 0), created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (service_provider_type_id, service_provider_topic_id), UNIQUE (service_provider_type_id, ordinal)
);
CREATE TABLE service_provider_unit_responsibilities (
    service_provider_type_id text NOT NULL REFERENCES service_provider_types(id), organizational_unit_id text NOT NULL REFERENCES caa_organizational_units(id),
    relationship text NOT NULL CHECK (relationship IN ('PRIMARY', 'JOINT', 'CONSULTED')), approval_required boolean NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY (service_provider_type_id, organizational_unit_id)
);

CREATE TABLE regulated_targets (
    id text PRIMARY KEY, target_kind text NOT NULL CHECK (target_kind IN ('ORGANIZATION','PERSON','FACILITY','DEVICE','SYSTEM','ASSET','LOCATION')),
    organization_id text REFERENCES organizations(id), person_subject_id text REFERENCES identity_references(subject_id), owner_organization_id text REFERENCES organizations(id),
    external_identifier text, created_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((target_kind = 'ORGANIZATION' AND organization_id IS NOT NULL AND person_subject_id IS NULL AND owner_organization_id IS NULL AND external_identifier IS NULL)
        OR (target_kind = 'PERSON' AND organization_id IS NULL AND person_subject_id IS NOT NULL AND external_identifier IS NULL)
        OR (target_kind IN ('FACILITY','DEVICE','SYSTEM','ASSET','LOCATION') AND organization_id IS NULL AND person_subject_id IS NULL AND btrim(external_identifier) <> '')),
    UNIQUE (target_kind, external_identifier), UNIQUE (organization_id), UNIQUE (person_subject_id)
);
CREATE TABLE organization_service_provider_scopes (
    id text PRIMARY KEY, organization_id text NOT NULL REFERENCES organizations(id), service_provider_type_id text NOT NULL REFERENCES service_provider_types(id),
    authorization_identifier text NOT NULL CHECK (btrim(authorization_identifier) <> ''), certificate_identifier text,
    status text NOT NULL CHECK (status IN ('ACTIVE','SUSPENDED','REVOKED','EXPIRED')), effective_from date NOT NULL, effective_to date,
    operation_qualifiers jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(operation_qualifiers) = 'object'),
    activity_qualifiers jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(activity_qualifiers) = 'object'),
    primary_target_id text REFERENCES regulated_targets(id), created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (effective_to IS NULL OR effective_to > effective_from), UNIQUE (organization_id, service_provider_type_id, authorization_identifier, effective_from)
);
CREATE UNIQUE INDEX organization_service_provider_scopes_active_identity_idx ON organization_service_provider_scopes (organization_id, service_provider_type_id, authorization_identifier) WHERE status = 'ACTIVE';
CREATE INDEX organization_service_provider_scope_applicability_idx ON organization_service_provider_scopes (organization_id, service_provider_type_id, effective_from, effective_to, id) WHERE status = 'ACTIVE';
CREATE TABLE organization_service_provider_scope_targets (
    organization_service_provider_scope_id text NOT NULL REFERENCES organization_service_provider_scopes(id), regulated_target_id text NOT NULL REFERENCES regulated_targets(id),
    created_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY (organization_service_provider_scope_id, regulated_target_id)
);

CREATE OR REPLACE FUNCTION governed_append_only_guard() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION '% records are append-only', TG_TABLE_NAME; END; $$;
CREATE OR REPLACE FUNCTION validate_governed_scope_target() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE allowed text[]; actual text;
BEGIN
 SELECT provider.target_kinds, target.target_kind INTO allowed, actual FROM organization_service_provider_scopes scope JOIN service_provider_types provider ON provider.id = scope.service_provider_type_id JOIN regulated_targets target ON target.id = NEW.regulated_target_id WHERE scope.id = NEW.organization_service_provider_scope_id;
 IF allowed IS NULL OR actual IS NULL OR NOT actual = ANY (allowed) THEN RAISE EXCEPTION 'regulated target kind is not compatible with service provider scope'; END IF;
 RETURN NEW;
END; $$;
CREATE OR REPLACE FUNCTION validate_governed_scope_primary_target() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE allowed text[]; actual text;
BEGIN
 IF NEW.primary_target_id IS NULL THEN RETURN NEW; END IF;
 SELECT provider.target_kinds, target.target_kind INTO allowed, actual FROM service_provider_types provider JOIN regulated_targets target ON target.id = NEW.primary_target_id WHERE provider.id = NEW.service_provider_type_id;
 IF allowed IS NULL OR actual IS NULL OR NOT actual = ANY (allowed) THEN RAISE EXCEPTION 'primary target kind is not compatible with service provider scope'; END IF;
 RETURN NEW;
END; $$;
CREATE TRIGGER organization_service_provider_scope_target_kind_guard BEFORE INSERT ON organization_service_provider_scope_targets FOR EACH ROW EXECUTE FUNCTION validate_governed_scope_target();
CREATE TRIGGER organization_service_provider_scope_primary_target_kind_guard BEFORE INSERT OR UPDATE OF primary_target_id, service_provider_type_id ON organization_service_provider_scopes FOR EACH ROW EXECUTE FUNCTION validate_governed_scope_primary_target();
CREATE TRIGGER caa_department_memberships_append_only BEFORE UPDATE OR DELETE ON caa_department_memberships FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER service_provider_types_append_only BEFORE UPDATE OR DELETE ON service_provider_types FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER service_provider_topics_append_only BEFORE UPDATE OR DELETE ON service_provider_topics FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER service_provider_topic_links_append_only BEFORE UPDATE OR DELETE ON service_provider_topic_links FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER service_provider_unit_responsibilities_append_only BEFORE UPDATE OR DELETE ON service_provider_unit_responsibilities FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER regulated_targets_append_only BEFORE UPDATE OR DELETE ON regulated_targets FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER organization_service_provider_scopes_append_only BEFORE UPDATE OR DELETE ON organization_service_provider_scopes FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER organization_service_provider_scope_targets_append_only BEFORE UPDATE OR DELETE ON organization_service_provider_scope_targets FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();

INSERT INTO caa_departments (id, name, status) VALUES
 ('FLIGHT_OPERATIONS_INSPECTORATE','Flight Operations Inspectorate (FOI)','ACTIVE'),('AIRWORTHINESS_INSPECTORATE','Airworthiness Inspectorate','ACTIVE'),('PERSONNEL_LICENSING_AND_TRAINING_DEPARTMENT','Personnel Licensing & Training Department','ACTIVE'),('AERODROME_INSPECTORATE','Aerodrome Inspectorate','ACTIVE'),('AIR_NAVIGATION_SERVICES_INSPECTORATE','Air Navigation Services Inspectorate','ACTIVE'),('CNS_INSPECTORATE','CNS Inspectorate','ACTIVE'),('METEOROLOGICAL_OVERSIGHT_UNIT','Meteorological Oversight Unit','ACTIVE'),('SAR_OVERSIGHT_UNIT','SAR Oversight Unit','ACTIVE'),('AVSEC_INSPECTORATE','AVSEC Inspectorate','ACTIVE'),('AIRWORTHINESS_CERTIFICATION_DEPARTMENT','Airworthiness Certification Department','ACTIVE'),('AEROMEDICAL_DEPARTMENT','Aeromedical Department','ACTIVE');
INSERT INTO caa_organizational_units (id, department_id, name, status) SELECT id, id, name, status FROM caa_departments;
INSERT INTO service_provider_types (id,catalog_version,label,raw_oversight_topics,raw_responsible_caa_unit,target_kinds,normalization_status) VALUES
 ('AIR_OPERATOR','1.0.0','Air Operator (AOC Holder)','Flight Operations; Cabin Safety; Operational Control; Crew Training; Dangerous Goods; SMS; Security; Manuals','Flight Operations Inspectorate (FOI)',ARRAY['ORGANIZATION'],'NORMALIZED'),
 ('AMO','1.0.0','Approved Maintenance Organization (AMO)','Maintenance Procedures; Personnel; Tooling; Facilities; Quality System; SMS; Records','Airworthiness Inspectorate',ARRAY['ORGANIZATION','FACILITY'],'NORMALIZED'),
 ('CAMO','1.0.0','Continuing Airworthiness Management Organization (CAMO)','Airworthiness Management; Maintenance Programme; Reliability Programme; Technical Records','Airworthiness Inspectorate',ARRAY['ORGANIZATION'],'NORMALIZED'),
 ('ATO','1.0.0','Approved Training Organization (ATO)','Training Programmes; Instructors; Examinations; Training Records; Simulators','Personnel Licensing & Training Department',ARRAY['ORGANIZATION','FACILITY'],'NORMALIZED'),
 ('FSTD','1.0.0','Flight Simulation Training Device (FSTD)','Simulator Qualification; Configuration; Maintenance; Records','Personnel Licensing & Training Department',ARRAY['DEVICE','FACILITY'],'NORMALIZED'),
 ('AERODROME_OPERATOR','1.0.0','Aerodrome Operator','Runways; Taxiways; Aprons; Lighting; RFFS; Wildlife; Obstacle Control; Emergency Plan; SMS','Aerodrome Inspectorate',ARRAY['ORGANIZATION','FACILITY','LOCATION'],'NORMALIZED'),
 ('ANSP','1.0.0','Air Navigation Service Provider (ANSP)','ATS; ATC Procedures; Airspace Management; SMS; Contingency Plans','Air Navigation Services Inspectorate',ARRAY['ORGANIZATION','SYSTEM'],'NORMALIZED'),
 ('CNS_PROVIDER','1.0.0','Communication Service Provider (CNS)','Communication Systems; Navigation Aids; Surveillance Systems; Maintenance','CNS Inspectorate',ARRAY['ORGANIZATION','SYSTEM','ASSET'],'NORMALIZED'),
 ('AIS_AIM_PROVIDER','1.0.0','AIS/AIM Provider','AIP; NOTAM; Charts; Data Quality; Digital AIM','AIS/AIM Inspectorate',ARRAY['ORGANIZATION','SYSTEM'],'REVIEW_REQUIRED'),
 ('MET_PROVIDER','1.0.0','Meteorological Service Provider (MET)','Aviation Weather Services; Forecasting; Observations; MET Reports','Meteorological Oversight Unit',ARRAY['ORGANIZATION','SYSTEM','FACILITY'],'NORMALIZED'),
 ('SAR_ORGANIZATION','1.0.0','Search and Rescue (SAR) Organization','Rescue Coordination; SAR Plans; Exercises; Readiness','SAR Oversight Unit',ARRAY['ORGANIZATION','FACILITY','LOCATION'],'NORMALIZED'),
 ('GROUND_HANDLING','1.0.0','Ground Handling Organization','Passenger Handling; Ramp Operations; Load Control; Baggage; Aircraft Servicing; SMS','Ground Operations / Flight Operations Inspectorate',ARRAY['ORGANIZATION','FACILITY'],'REVIEW_REQUIRED'),
 ('FUEL_PROVIDER','1.0.0','Fuel Service Provider','Fuel Storage; Fuel Quality; Refuelling Procedures; Equipment; Personnel','Aerodrome Inspectorate',ARRAY['ORGANIZATION','FACILITY','ASSET','LOCATION'],'NORMALIZED'),
 ('CARGO_REGULATED_AGENT','1.0.0','Cargo Terminal / Regulated Agent','Cargo Acceptance; Dangerous Goods; Security Controls; Documentation','Aviation Security (AVSEC) + Dangerous Goods Office',ARRAY['ORGANIZATION','FACILITY'],'REVIEW_REQUIRED'),
 ('AVSEC_PROVIDER','1.0.0','Aviation Security Service Provider','Passenger Screening; Access Control; Hold Baggage Screening; Staff Training','AVSEC Inspectorate',ARRAY['ORGANIZATION','FACILITY'],'NORMALIZED'),
 ('RPAS_UAS_OPERATOR','1.0.0','RPAS/UAS Operator','Flight Operations; Remote Pilot Competency; Maintenance; Operational Risk Assessment; C2 Link','RPAS / Flight Operations Inspectorate',ARRAY['ORGANIZATION','DEVICE','SYSTEM'],'REVIEW_REQUIRED'),
 ('DOA','1.0.0','Aircraft Design Organization (DOA)','Design Approval; Compliance Demonstration; Configuration Control','Airworthiness Certification Department',ARRAY['ORGANIZATION'],'NORMALIZED'),
 ('POA','1.0.0','Production Organization (POA)','Production System; Quality Assurance; Product Conformity','Airworthiness Certification Department',ARRAY['ORGANIZATION','FACILITY'],'NORMALIZED'),
 ('AEMC','1.0.0','Aviation Medical Centre (AeMC)','Medical Facilities; Equipment; Medical Records; Personnel','Aeromedical Department',ARRAY['ORGANIZATION','FACILITY'],'NORMALIZED'),
 ('AME','1.0.0','Aviation Medical Examiner (AME)','Medical Examinations; Certification Procedures; Record Keeping','Aeromedical Department',ARRAY['PERSON'],'NORMALIZED');
INSERT INTO service_provider_topics (id, topic) SELECT topic, topic FROM (SELECT DISTINCT btrim(topic) topic FROM service_provider_types, LATERAL regexp_split_to_table(raw_oversight_topics, ';') topic) AS topics;
INSERT INTO service_provider_topic_links (service_provider_type_id,service_provider_topic_id,ordinal) SELECT provider.id,btrim(topic),ordinality FROM service_provider_types provider CROSS JOIN LATERAL regexp_split_to_table(provider.raw_oversight_topics,';') WITH ORDINALITY AS parts(topic,ordinality);
INSERT INTO service_provider_unit_responsibilities (service_provider_type_id,organizational_unit_id,relationship,approval_required) VALUES
 ('AIR_OPERATOR','FLIGHT_OPERATIONS_INSPECTORATE','PRIMARY',true),('AMO','AIRWORTHINESS_INSPECTORATE','PRIMARY',true),('CAMO','AIRWORTHINESS_INSPECTORATE','PRIMARY',true),('ATO','PERSONNEL_LICENSING_AND_TRAINING_DEPARTMENT','PRIMARY',true),('FSTD','PERSONNEL_LICENSING_AND_TRAINING_DEPARTMENT','PRIMARY',true),('AERODROME_OPERATOR','AERODROME_INSPECTORATE','PRIMARY',true),('ANSP','AIR_NAVIGATION_SERVICES_INSPECTORATE','PRIMARY',true),('CNS_PROVIDER','CNS_INSPECTORATE','PRIMARY',true),('MET_PROVIDER','METEOROLOGICAL_OVERSIGHT_UNIT','PRIMARY',true),('SAR_ORGANIZATION','SAR_OVERSIGHT_UNIT','PRIMARY',true),('FUEL_PROVIDER','AERODROME_INSPECTORATE','PRIMARY',true),('AVSEC_PROVIDER','AVSEC_INSPECTORATE','PRIMARY',true),('DOA','AIRWORTHINESS_CERTIFICATION_DEPARTMENT','PRIMARY',true),('POA','AIRWORTHINESS_CERTIFICATION_DEPARTMENT','PRIMARY',true),('AEMC','AEROMEDICAL_DEPARTMENT','PRIMARY',true),('AME','AEROMEDICAL_DEPARTMENT','PRIMARY',true);

ALTER TABLE caa_departments ADD COLUMN baseline_seed boolean NOT NULL DEFAULT false;
ALTER TABLE caa_organizational_units ADD COLUMN baseline_seed boolean NOT NULL DEFAULT false;
ALTER TABLE service_provider_types ADD COLUMN baseline_seed boolean NOT NULL DEFAULT false;
ALTER TABLE service_provider_topics ADD COLUMN baseline_seed boolean NOT NULL DEFAULT false;
ALTER TABLE service_provider_topic_links ADD COLUMN baseline_seed boolean NOT NULL DEFAULT false;
ALTER TABLE service_provider_unit_responsibilities ADD COLUMN baseline_seed boolean NOT NULL DEFAULT false;
UPDATE caa_departments SET baseline_seed = true;
UPDATE caa_organizational_units SET baseline_seed = true;
ALTER TABLE service_provider_types DISABLE TRIGGER service_provider_types_append_only; UPDATE service_provider_types SET baseline_seed = true; ALTER TABLE service_provider_types ENABLE TRIGGER service_provider_types_append_only;
ALTER TABLE service_provider_topics DISABLE TRIGGER service_provider_topics_append_only; UPDATE service_provider_topics SET baseline_seed = true; ALTER TABLE service_provider_topics ENABLE TRIGGER service_provider_topics_append_only;
ALTER TABLE service_provider_topic_links DISABLE TRIGGER service_provider_topic_links_append_only; UPDATE service_provider_topic_links SET baseline_seed = true; ALTER TABLE service_provider_topic_links ENABLE TRIGGER service_provider_topic_links_append_only;
ALTER TABLE service_provider_unit_responsibilities DISABLE TRIGGER service_provider_unit_responsibilities_append_only; UPDATE service_provider_unit_responsibilities SET baseline_seed = true; ALTER TABLE service_provider_unit_responsibilities ENABLE TRIGGER service_provider_unit_responsibilities_append_only;
CREATE OR REPLACE FUNCTION governed_baseline_seed_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.baseline_seed THEN
  RAISE EXCEPTION 'baseline_seed is reserved for migration-owned catalog facts';
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER caa_departments_baseline_seed_guard BEFORE INSERT ON caa_departments FOR EACH ROW EXECUTE FUNCTION governed_baseline_seed_guard();
CREATE TRIGGER caa_organizational_units_baseline_seed_guard BEFORE INSERT ON caa_organizational_units FOR EACH ROW EXECUTE FUNCTION governed_baseline_seed_guard();
CREATE TRIGGER service_provider_types_baseline_seed_guard BEFORE INSERT ON service_provider_types FOR EACH ROW EXECUTE FUNCTION governed_baseline_seed_guard();
CREATE TRIGGER service_provider_topics_baseline_seed_guard BEFORE INSERT ON service_provider_topics FOR EACH ROW EXECUTE FUNCTION governed_baseline_seed_guard();
CREATE TRIGGER service_provider_topic_links_baseline_seed_guard BEFORE INSERT ON service_provider_topic_links FOR EACH ROW EXECUTE FUNCTION governed_baseline_seed_guard();
CREATE TRIGGER service_provider_unit_responsibilities_baseline_seed_guard BEFORE INSERT ON service_provider_unit_responsibilities FOR EACH ROW EXECUTE FUNCTION governed_baseline_seed_guard();
CREATE TRIGGER caa_departments_append_only BEFORE UPDATE OR DELETE ON caa_departments FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER caa_organizational_units_append_only BEFORE UPDATE OR DELETE ON caa_organizational_units FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
ALTER TABLE caa_organizational_units ADD CONSTRAINT caa_units_department_id_id_unique UNIQUE (department_id, id);
ALTER TABLE caa_department_memberships ADD CONSTRAINT caa_memberships_department_unit_fk FOREIGN KEY (department_id, organizational_unit_id) REFERENCES caa_organizational_units(department_id, id);
ALTER TABLE caa_department_memberships ALTER COLUMN organizational_unit_id SET NOT NULL;
ALTER TABLE caa_department_memberships ADD COLUMN root_id text; ALTER TABLE caa_department_memberships ADD COLUMN supersedes_id text UNIQUE REFERENCES caa_department_memberships(id); UPDATE caa_department_memberships SET root_id=id; ALTER TABLE caa_department_memberships ALTER COLUMN root_id SET NOT NULL; ALTER TABLE caa_department_memberships ADD CONSTRAINT caa_memberships_root_fk FOREIGN KEY (root_id) REFERENCES caa_department_memberships(id);
CREATE UNIQUE INDEX caa_department_membership_root_identity_idx ON caa_department_memberships (subject_id, department_id, organizational_unit_id) WHERE supersedes_id IS NULL;
DROP INDEX caa_department_memberships_effective_authority_idx;
CREATE INDEX caa_department_memberships_effective_authority_idx ON caa_department_memberships (subject_id, root_id, effective_from DESC, id DESC);
DROP INDEX organization_service_provider_scopes_active_identity_idx;
ALTER TABLE organization_service_provider_scopes ADD COLUMN root_id text; ALTER TABLE organization_service_provider_scopes ADD COLUMN supersedes_id text UNIQUE REFERENCES organization_service_provider_scopes(id); UPDATE organization_service_provider_scopes SET root_id=id; ALTER TABLE organization_service_provider_scopes ALTER COLUMN root_id SET NOT NULL; ALTER TABLE organization_service_provider_scopes ADD CONSTRAINT organization_scopes_root_fk FOREIGN KEY (root_id) REFERENCES organization_service_provider_scopes(id);
CREATE UNIQUE INDEX organization_service_provider_scope_root_identity_idx ON organization_service_provider_scopes (organization_id, service_provider_type_id, authorization_identifier) WHERE supersedes_id IS NULL;
DROP INDEX organization_service_provider_scope_applicability_idx;
CREATE INDEX organization_service_provider_scope_applicability_idx ON organization_service_provider_scopes (organization_id, root_id, effective_from DESC, id DESC);
CREATE OR REPLACE FUNCTION governed_successor_guard() RETURNS trigger LANGUAGE plpgsql AS $$ DECLARE p record; BEGIN IF NEW.supersedes_id IS NULL THEN NEW.root_id:=NEW.id; RETURN NEW; END IF; IF TG_TABLE_NAME='organization_service_provider_scopes' THEN SELECT * INTO p FROM organization_service_provider_scopes WHERE id=NEW.supersedes_id; IF p.id IS NULL OR p.root_id<>NEW.root_id OR p.organization_id<>NEW.organization_id OR p.service_provider_type_id<>NEW.service_provider_type_id OR p.authorization_identifier<>NEW.authorization_identifier OR NEW.effective_from<=p.effective_from THEN RAISE EXCEPTION 'invalid governed scope successor'; END IF; ELSE SELECT * INTO p FROM caa_department_memberships WHERE id=NEW.supersedes_id; IF p.id IS NULL OR p.root_id<>NEW.root_id OR p.subject_id<>NEW.subject_id OR p.department_id<>NEW.department_id OR p.organizational_unit_id<>NEW.organizational_unit_id OR NEW.effective_from<=p.effective_from THEN RAISE EXCEPTION 'invalid department membership successor'; END IF; END IF; RETURN NEW; END; $$;
CREATE TRIGGER organization_scope_successor_guard BEFORE INSERT ON organization_service_provider_scopes FOR EACH ROW EXECUTE FUNCTION governed_successor_guard();
CREATE TRIGGER department_membership_successor_guard BEFORE INSERT ON caa_department_memberships FOR EACH ROW EXECUTE FUNCTION governed_successor_guard();
CREATE TABLE caa_department_status_facts (
    id text PRIMARY KEY, root_id text NOT NULL, supersedes_id text UNIQUE REFERENCES caa_department_status_facts(id),
    department_id text NOT NULL REFERENCES caa_departments(id), status text NOT NULL CHECK (status IN ('ACTIVE', 'INACTIVE')),
    effective_from date NOT NULL, baseline_seed boolean NOT NULL DEFAULT false, created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (root_id) REFERENCES caa_department_status_facts(id)
);
CREATE TABLE caa_organizational_unit_status_facts (
    id text PRIMARY KEY, root_id text NOT NULL, supersedes_id text UNIQUE REFERENCES caa_organizational_unit_status_facts(id),
    organizational_unit_id text NOT NULL REFERENCES caa_organizational_units(id), status text NOT NULL CHECK (status IN ('ACTIVE', 'INACTIVE')),
    effective_from date NOT NULL, baseline_seed boolean NOT NULL DEFAULT false, created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (root_id) REFERENCES caa_organizational_unit_status_facts(id)
);
CREATE UNIQUE INDEX caa_department_status_fact_root_identity_idx ON caa_department_status_facts (department_id) WHERE supersedes_id IS NULL;
CREATE UNIQUE INDEX caa_organizational_unit_status_fact_root_identity_idx ON caa_organizational_unit_status_facts (organizational_unit_id) WHERE supersedes_id IS NULL;
CREATE INDEX caa_department_status_facts_effective_idx ON caa_department_status_facts (department_id, root_id, effective_from DESC, id DESC);
CREATE INDEX caa_organizational_unit_status_facts_effective_idx ON caa_organizational_unit_status_facts (organizational_unit_id, root_id, effective_from DESC, id DESC);
CREATE OR REPLACE FUNCTION governed_status_successor_guard() RETURNS trigger LANGUAGE plpgsql AS $$ DECLARE p record; BEGIN IF NEW.supersedes_id IS NULL THEN NEW.root_id := NEW.id; RETURN NEW; END IF; IF TG_TABLE_NAME = 'caa_department_status_facts' THEN SELECT * INTO p FROM caa_department_status_facts WHERE id = NEW.supersedes_id; IF p.id IS NULL OR p.root_id <> NEW.root_id OR p.department_id <> NEW.department_id OR NEW.effective_from <= p.effective_from THEN RAISE EXCEPTION 'invalid department status successor'; END IF; ELSE SELECT * INTO p FROM caa_organizational_unit_status_facts WHERE id = NEW.supersedes_id; IF p.id IS NULL OR p.root_id <> NEW.root_id OR p.organizational_unit_id <> NEW.organizational_unit_id OR NEW.effective_from <= p.effective_from THEN RAISE EXCEPTION 'invalid organizational unit status successor'; END IF; END IF; RETURN NEW; END; $$;
CREATE TRIGGER caa_department_status_successor_guard BEFORE INSERT ON caa_department_status_facts FOR EACH ROW EXECUTE FUNCTION governed_status_successor_guard();
CREATE TRIGGER caa_organizational_unit_status_successor_guard BEFORE INSERT ON caa_organizational_unit_status_facts FOR EACH ROW EXECUTE FUNCTION governed_status_successor_guard();
CREATE TRIGGER caa_department_status_baseline_seed_guard BEFORE INSERT ON caa_department_status_facts FOR EACH ROW EXECUTE FUNCTION governed_baseline_seed_guard();
CREATE TRIGGER caa_organizational_unit_status_baseline_seed_guard BEFORE INSERT ON caa_organizational_unit_status_facts FOR EACH ROW EXECUTE FUNCTION governed_baseline_seed_guard();
CREATE TRIGGER caa_department_status_facts_append_only BEFORE UPDATE OR DELETE ON caa_department_status_facts FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER caa_organizational_unit_status_facts_append_only BEFORE UPDATE OR DELETE ON caa_organizational_unit_status_facts FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
ALTER TABLE caa_department_status_facts DISABLE TRIGGER caa_department_status_baseline_seed_guard;
ALTER TABLE caa_organizational_unit_status_facts DISABLE TRIGGER caa_organizational_unit_status_baseline_seed_guard;
INSERT INTO caa_department_status_facts (id, root_id, department_id, status, effective_from, baseline_seed) SELECT 'seed-department-status-' || id, 'seed-department-status-' || id, id, 'ACTIVE', '0001-01-01', true FROM caa_departments;
INSERT INTO caa_organizational_unit_status_facts (id, root_id, organizational_unit_id, status, effective_from, baseline_seed) SELECT 'seed-unit-status-' || id, 'seed-unit-status-' || id, id, 'ACTIVE', '0001-01-01', true FROM caa_organizational_units;
ALTER TABLE caa_department_status_facts ENABLE TRIGGER caa_department_status_baseline_seed_guard;
ALTER TABLE caa_organizational_unit_status_facts ENABLE TRIGGER caa_organizational_unit_status_baseline_seed_guard;
CREATE OR REPLACE FUNCTION validate_governed_scope_target() RETURNS trigger LANGUAGE plpgsql AS $$ DECLARE allowed text[]; actual text; scope_org text; target_org text; BEGIN SELECT p.target_kinds,t.target_kind,s.organization_id,COALESCE(t.organization_id,t.owner_organization_id) INTO allowed,actual,scope_org,target_org FROM organization_service_provider_scopes s JOIN service_provider_types p ON p.id=s.service_provider_type_id JOIN regulated_targets t ON t.id=NEW.regulated_target_id WHERE s.id=NEW.organization_service_provider_scope_id; IF allowed IS NULL OR NOT actual=ANY(allowed) OR (target_org IS NOT NULL AND target_org<>scope_org) THEN RAISE EXCEPTION 'regulated target is not compatible with scope identity'; END IF; RETURN NEW; END; $$;
CREATE OR REPLACE FUNCTION validate_governed_scope_primary_target() RETURNS trigger LANGUAGE plpgsql AS $$ DECLARE allowed text[]; actual text; target_org text; BEGIN IF NEW.primary_target_id IS NULL THEN RETURN NEW; END IF; SELECT p.target_kinds,t.target_kind,COALESCE(t.organization_id,t.owner_organization_id) INTO allowed,actual,target_org FROM service_provider_types p JOIN regulated_targets t ON t.id=NEW.primary_target_id WHERE p.id=NEW.service_provider_type_id; IF allowed IS NULL OR NOT actual=ANY(allowed) OR (target_org IS NOT NULL AND target_org<>NEW.organization_id) THEN RAISE EXCEPTION 'primary target is not compatible with scope identity'; END IF; RETURN NEW; END; $$;

-- Task 3: source identities and clause locators retain only bounded metadata,
-- hashes, and locators. Regulatory source text remains outside Git.
CREATE OR REPLACE FUNCTION governed_sha256(value text) RETURNS boolean LANGUAGE sql IMMUTABLE AS $$
    SELECT value ~ '^sha256:[0-9a-f]{64}$'
$$;

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE OR REPLACE FUNCTION governed_jsonb_sha256(value jsonb) RETURNS text LANGUAGE sql IMMUTABLE STRICT AS $$
    SELECT 'sha256:' || encode(digest(convert_to(value::text, 'UTF8'), 'sha256'), 'hex')
$$;

CREATE TABLE regulatory_source_versions (
    id text PRIMARY KEY,
    source_identity text NOT NULL CHECK (btrim(source_identity) <> ''),
    version_identity text NOT NULL CHECK (btrim(version_identity) <> ''),
    title text NOT NULL CHECK (btrim(title) <> ''),
    source_class text NOT NULL CHECK (source_class IN ('PRIMARY_AUTHORITY','STATE_COMPLIANCE_CROSSWALK','CONTROLLED_PROCEDURE','DERIVED_CONTEXT')),
    source_status text NOT NULL CHECK (source_status IN ('SUPPLIED_WORKING_COPY','PUBLIC_REFERENCE','SOURCE_GAP')),
    source_locator text NOT NULL CHECK (btrim(source_locator) <> ''),
    source_url text,
    source_hash text NOT NULL CHECK (governed_sha256(source_hash)),
    effective_from date NOT NULL,
    effective_to date,
    source_metadata jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(source_metadata) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (effective_to IS NULL OR effective_to > effective_from),
    UNIQUE (source_identity, version_identity)
);
CREATE TABLE regulatory_normalized_clauses (
    id text PRIMARY KEY,
    regulatory_source_version_id text NOT NULL REFERENCES regulatory_source_versions(id),
    clause_identity text NOT NULL CHECK (btrim(clause_identity) <> ''),
    annex_identity text NOT NULL CHECK (btrim(annex_identity) <> ''),
    section_identity text NOT NULL CHECK (btrim(section_identity) <> ''),
    clause_locator text NOT NULL CHECK (btrim(clause_locator) <> ''),
    source_hash text NOT NULL CHECK (governed_sha256(source_hash)),
    normalized_digest text NOT NULL CHECK (governed_sha256(normalized_digest)),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (regulatory_source_version_id, clause_identity)
);
CREATE TABLE state_compliance_crosswalk_rows (
    id text PRIMARY KEY,
    regulatory_source_version_id text NOT NULL REFERENCES regulatory_source_versions(id),
    normalized_clause_id text NOT NULL REFERENCES regulatory_normalized_clauses(id),
    stable_row_identity text NOT NULL CHECK (btrim(stable_row_identity) <> ''),
    annex_identity text NOT NULL CHECK (btrim(annex_identity) <> ''),
    section_identity text NOT NULL CHECK (btrim(section_identity) <> ''),
    row_digest text NOT NULL CHECK (governed_sha256(row_digest)),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (regulatory_source_version_id, stable_row_identity)
);
CREATE INDEX regulatory_normalized_clauses_source_locator_idx ON regulatory_normalized_clauses (regulatory_source_version_id, annex_identity, section_identity, clause_identity);

CREATE OR REPLACE FUNCTION validate_governed_clause_source_hash() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE expected_hash text;
BEGIN
    SELECT source_hash INTO expected_hash FROM regulatory_source_versions WHERE id = NEW.regulatory_source_version_id;
    IF expected_hash IS NULL OR NEW.source_hash <> expected_hash THEN
        RAISE EXCEPTION 'normalized clause source hash does not match its source version';
    END IF;
    RETURN NEW;
END;
$$;
CREATE OR REPLACE FUNCTION validate_governed_crosswalk_row() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE actual_source_class text; actual_source_status text; clause_source_id text; clause_annex text; clause_section text;
BEGIN
    SELECT version.source_class, version.source_status INTO actual_source_class, actual_source_status FROM regulatory_source_versions version WHERE version.id = NEW.regulatory_source_version_id;
    SELECT regulatory_source_version_id, annex_identity, section_identity INTO clause_source_id, clause_annex, clause_section FROM regulatory_normalized_clauses WHERE id = NEW.normalized_clause_id;
    IF actual_source_class <> 'STATE_COMPLIANCE_CROSSWALK' OR actual_source_status <> 'SUPPLIED_WORKING_COPY' OR clause_source_id <> NEW.regulatory_source_version_id OR clause_annex <> NEW.annex_identity OR clause_section <> NEW.section_identity THEN
        RAISE EXCEPTION 'crosswalk rows require one supplied working-copy State crosswalk source and matching normalized clause identity';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TABLE regulatory_evaluations (
    id text PRIMARY KEY,
    evaluation_identity text NOT NULL UNIQUE CHECK (btrim(evaluation_identity) <> ''),
    purpose text NOT NULL CHECK (btrim(purpose) <> ''),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE regulatory_evaluation_partitions (
    id text PRIMARY KEY,
    evaluation_id text NOT NULL REFERENCES regulatory_evaluations(id),
    partition_kind text NOT NULL CHECK (partition_kind IN ('GENERATION_INPUT','BLIND_HOLDOUT')),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (evaluation_id, partition_kind),
    UNIQUE (id, evaluation_id)
);
CREATE TABLE regulatory_evaluation_partition_rows (
    evaluation_id text NOT NULL,
    partition_id text NOT NULL,
    state_compliance_crosswalk_row_id text NOT NULL REFERENCES state_compliance_crosswalk_rows(id),
    stable_row_identity text NOT NULL CHECK (btrim(stable_row_identity) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (partition_id, state_compliance_crosswalk_row_id),
    UNIQUE (evaluation_id, stable_row_identity),
    FOREIGN KEY (partition_id, evaluation_id) REFERENCES regulatory_evaluation_partitions(id, evaluation_id)
);
CREATE OR REPLACE FUNCTION validate_governed_partition_identity() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE expected_identity text;
BEGIN
    SELECT stable_row_identity INTO expected_identity FROM state_compliance_crosswalk_rows WHERE id = NEW.state_compliance_crosswalk_row_id;
    IF expected_identity IS NULL OR expected_identity <> NEW.stable_row_identity THEN
        RAISE EXCEPTION 'evaluation partition identity must match the immutable supplied CC row identity';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TABLE regulatory_generation_runs (
    id text PRIMARY KEY,
    status text NOT NULL CHECK (status IN ('GENERATED','FAILED')),
    input_digest text NOT NULL UNIQUE CHECK (input_digest = governed_jsonb_sha256(input_artifact)),
    output_digest text CHECK (output_digest IS NULL OR output_digest = governed_jsonb_sha256(output_artifact)),
    input_schema_version text NOT NULL CHECK (btrim(input_schema_version) <> ''),
    generation_policy_version text NOT NULL CHECK (btrim(generation_policy_version) <> ''),
    provider_catalog_version text NOT NULL CHECK (btrim(provider_catalog_version) <> ''),
    provider_adapter_version text NOT NULL CHECK (btrim(provider_adapter_version) <> ''),
    inspection_type text NOT NULL CHECK (btrim(inspection_type) <> ''),
    target_id text NOT NULL REFERENCES regulated_targets(id),
    input_artifact jsonb NOT NULL CHECK (jsonb_typeof(input_artifact) = 'object'),
    output_artifact jsonb CHECK (output_artifact IS NULL OR jsonb_typeof(output_artifact) = 'object'),
    failure_code text,
    failure_reason text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((status = 'GENERATED' AND output_digest IS NOT NULL AND output_artifact IS NOT NULL AND failure_code IS NULL AND failure_reason IS NULL) OR (status = 'FAILED' AND output_digest IS NULL AND output_artifact IS NULL AND failure_reason IS NOT NULL))
);
CREATE TABLE regulatory_generation_run_source_snapshots (
    generation_run_id text NOT NULL REFERENCES regulatory_generation_runs(id),
    regulatory_source_version_id text NOT NULL REFERENCES regulatory_source_versions(id),
    regulatory_normalized_clause_id text NOT NULL REFERENCES regulatory_normalized_clauses(id),
    source_hash text NOT NULL CHECK (governed_sha256(source_hash)),
    clause_locator text NOT NULL CHECK (btrim(clause_locator) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (generation_run_id, regulatory_normalized_clause_id)
);
CREATE OR REPLACE FUNCTION validate_governed_generation_snapshot() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE expected_hash text; clause_source text; expected_locator text;
BEGIN
    SELECT source_hash INTO expected_hash FROM regulatory_source_versions WHERE id = NEW.regulatory_source_version_id;
    SELECT regulatory_source_version_id, clause_locator INTO clause_source, expected_locator FROM regulatory_normalized_clauses WHERE id = NEW.regulatory_normalized_clause_id;
    IF expected_hash IS NULL OR clause_source <> NEW.regulatory_source_version_id OR NEW.source_hash <> expected_hash OR NEW.clause_locator <> expected_locator THEN
        RAISE EXCEPTION 'generation source snapshot must pin its exact source hash and normalized clause locator';
    END IF;
    RETURN NEW;
END;
$$;

ALTER TABLE template_draft_versions DROP CONSTRAINT template_draft_versions_status_check;
ALTER TABLE template_draft_versions ADD CONSTRAINT template_draft_versions_status_check CHECK (status IN ('DRAFT','GENERATED_DRAFT','DEPARTMENT_REVIEW','RETURNED','REJECTED','TECHNICALLY_APPROVED','PUBLISHED'));
ALTER TABLE template_draft_versions ADD COLUMN generation_run_id text REFERENCES regulatory_generation_runs(id);
ALTER TABLE template_draft_versions ADD COLUMN candidate_content_digest text CHECK (candidate_content_digest IS NULL OR governed_sha256(candidate_content_digest));
ALTER TABLE template_draft_versions ADD COLUMN candidate_schema_version text;
ALTER TABLE template_draft_versions ADD CONSTRAINT template_draft_versions_generated_lineage_check CHECK ((generation_run_id IS NULL AND candidate_content_digest IS NULL AND candidate_schema_version IS NULL) OR (generation_run_id IS NOT NULL AND candidate_content_digest IS NOT NULL AND candidate_schema_version IS NOT NULL));
ALTER TABLE template_draft_versions ADD CONSTRAINT template_draft_versions_candidate_identity_unique UNIQUE (id, revision, candidate_content_digest);
CREATE OR REPLACE FUNCTION governed_generated_candidate_immutable_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.generation_run_id IS NOT NULL THEN RAISE EXCEPTION 'generated candidate revisions are immutable'; END IF;
    RETURN NEW;
END;
$$;

CREATE TABLE candidate_required_owner_assignments (
    id text PRIMARY KEY,
    candidate_draft_version_id text NOT NULL,
    candidate_revision bigint NOT NULL,
    candidate_content_digest text NOT NULL CHECK (governed_sha256(candidate_content_digest)),
    department_id text NOT NULL REFERENCES caa_departments(id),
    organizational_unit_id text NOT NULL,
    approval_required boolean NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (candidate_draft_version_id, candidate_revision, candidate_content_digest, department_id, organizational_unit_id),
    FOREIGN KEY (candidate_draft_version_id, candidate_revision, candidate_content_digest) REFERENCES template_draft_versions(id, revision, candidate_content_digest),
    FOREIGN KEY (department_id, organizational_unit_id) REFERENCES caa_organizational_units(department_id, id)
);
CREATE TABLE department_review_decisions (
    id text PRIMARY KEY,
    candidate_root_id text NOT NULL REFERENCES template_draft_versions(id),
    candidate_draft_version_id text NOT NULL,
    candidate_revision bigint NOT NULL,
    candidate_content_digest text NOT NULL CHECK (governed_sha256(candidate_content_digest)),
    decision text NOT NULL CHECK (decision IN ('RETURNED','REJECTED','TECHNICALLY_APPROVED')),
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    actor_department_membership_id text NOT NULL REFERENCES caa_department_memberships(id),
    actor_department_id text NOT NULL REFERENCES caa_departments(id),
    actor_organizational_unit_id text NOT NULL,
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    decided_at timestamptz NOT NULL,
    operation_id text NOT NULL UNIQUE CHECK (btrim(operation_id) <> ''),
    idempotency_key text NOT NULL UNIQUE CHECK (btrim(idempotency_key) <> ''),
    semantic_payload_digest text NOT NULL CHECK (governed_sha256(semantic_payload_digest)),
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (candidate_draft_version_id, candidate_revision, candidate_content_digest) REFERENCES template_draft_versions(id, revision, candidate_content_digest),
    FOREIGN KEY (actor_department_id, actor_organizational_unit_id) REFERENCES caa_organizational_units(department_id, id)
);
CREATE TABLE checklist_publication_decisions (
    id text PRIMARY KEY,
    candidate_root_id text NOT NULL REFERENCES template_draft_versions(id),
    candidate_draft_version_id text NOT NULL,
    candidate_revision bigint NOT NULL,
    candidate_content_digest text NOT NULL CHECK (governed_sha256(candidate_content_digest)),
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    actor_department_membership_id text NOT NULL REFERENCES caa_department_memberships(id),
    actor_department_id text NOT NULL REFERENCES caa_departments(id),
    actor_organizational_unit_id text NOT NULL,
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    decided_at timestamptz NOT NULL,
    operation_id text NOT NULL UNIQUE CHECK (btrim(operation_id) <> ''),
    idempotency_key text NOT NULL UNIQUE CHECK (btrim(idempotency_key) <> ''),
    semantic_payload_digest text NOT NULL CHECK (governed_sha256(semantic_payload_digest)),
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (candidate_draft_version_id, candidate_revision, candidate_content_digest) REFERENCES template_draft_versions(id, revision, candidate_content_digest),
    FOREIGN KEY (actor_department_id, actor_organizational_unit_id) REFERENCES caa_organizational_units(department_id, id)
);
CREATE INDEX candidate_required_owner_assignments_review_queue_idx
    ON candidate_required_owner_assignments
    (department_id, organizational_unit_id, candidate_draft_version_id, candidate_revision, candidate_content_digest)
    WHERE approval_required;
CREATE INDEX department_review_decisions_candidate_idx
    ON department_review_decisions
    (candidate_draft_version_id, candidate_revision, candidate_content_digest, decided_at, id);
CREATE UNIQUE INDEX department_review_decisions_exact_owner_approval_idx
    ON department_review_decisions
    (candidate_draft_version_id, candidate_revision, candidate_content_digest,
     actor_department_id, actor_organizational_unit_id)
    WHERE decision='TECHNICALLY_APPROVED';
CREATE UNIQUE INDEX checklist_publication_decisions_candidate_unique_idx
    ON checklist_publication_decisions
    (candidate_draft_version_id, candidate_revision, candidate_content_digest);
CREATE INDEX template_draft_versions_governed_review_queue_idx
    ON template_draft_versions (status, id)
    WHERE generation_run_id IS NOT NULL;
CREATE OR REPLACE FUNCTION validate_governed_decision_actor() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE membership record;
BEGIN
    SELECT * INTO membership FROM caa_department_memberships WHERE id = NEW.actor_department_membership_id;
    IF membership.id IS NULL OR membership.subject_id <> NEW.actor_subject_id OR membership.department_id <> NEW.actor_department_id OR membership.organizational_unit_id <> NEW.actor_organizational_unit_id OR membership.membership_role <> 'DEPARTMENT_MANAGER' OR membership.status <> 'ACTIVE' OR membership.effective_from > NEW.decided_at::date OR (membership.effective_to IS NOT NULL AND membership.effective_to <= NEW.decided_at::date) THEN
        RAISE EXCEPTION 'decision actor has no current matching Department Manager assignment';
    END IF;
    RETURN NEW;
END;
$$;
CREATE OR REPLACE FUNCTION validate_governed_publication_approval() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM candidate_required_owner_assignments WHERE candidate_draft_version_id = NEW.candidate_draft_version_id AND candidate_revision = NEW.candidate_revision AND candidate_content_digest = NEW.candidate_content_digest AND approval_required) OR EXISTS (
        SELECT 1 FROM candidate_required_owner_assignments owner WHERE owner.candidate_draft_version_id = NEW.candidate_draft_version_id AND owner.candidate_revision = NEW.candidate_revision AND owner.candidate_content_digest = NEW.candidate_content_digest AND owner.approval_required AND NOT EXISTS (
            SELECT 1 FROM department_review_decisions review WHERE review.candidate_draft_version_id = owner.candidate_draft_version_id AND review.candidate_revision = owner.candidate_revision AND review.candidate_content_digest = owner.candidate_content_digest AND review.decision = 'TECHNICALLY_APPROVED' AND review.actor_department_id = owner.department_id AND review.actor_organizational_unit_id = owner.organizational_unit_id
        )
    ) THEN RAISE EXCEPTION 'publication requires all required technical approvals for the exact candidate digest'; END IF;
    RETURN NEW;
END;
$$;
ALTER TABLE checklist_template_versions ADD COLUMN candidate_draft_version_id text;
ALTER TABLE checklist_template_versions ADD COLUMN candidate_revision bigint;
ALTER TABLE checklist_template_versions ADD COLUMN candidate_content_digest text CHECK (candidate_content_digest IS NULL OR governed_sha256(candidate_content_digest));
ALTER TABLE checklist_template_versions ADD COLUMN publication_decision_id text REFERENCES checklist_publication_decisions(id);
ALTER TABLE checklist_template_versions ADD CONSTRAINT checklist_template_versions_governed_publication_shape CHECK ((candidate_draft_version_id IS NULL AND candidate_revision IS NULL AND candidate_content_digest IS NULL AND publication_decision_id IS NULL) OR (candidate_draft_version_id IS NOT NULL AND candidate_revision IS NOT NULL AND candidate_content_digest IS NOT NULL AND publication_decision_id IS NOT NULL));
CREATE UNIQUE INDEX checklist_template_versions_governed_candidate_unique_idx
    ON checklist_template_versions
    (candidate_draft_version_id, candidate_revision, candidate_content_digest)
    WHERE candidate_draft_version_id IS NOT NULL;
CREATE OR REPLACE FUNCTION validate_governed_published_template() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE decision record;
BEGIN
    IF NEW.candidate_draft_version_id IS NULL THEN RETURN NEW; END IF;
    SELECT * INTO decision FROM checklist_publication_decisions WHERE id = NEW.publication_decision_id;
    IF decision.id IS NULL OR decision.candidate_draft_version_id <> NEW.candidate_draft_version_id OR decision.candidate_revision <> NEW.candidate_revision OR decision.candidate_content_digest <> NEW.candidate_content_digest THEN
        RAISE EXCEPTION 'published template must reference the separately recorded approved candidate digest';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER regulatory_source_versions_append_only BEFORE UPDATE OR DELETE ON regulatory_source_versions FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER regulatory_normalized_clauses_append_only BEFORE UPDATE OR DELETE ON regulatory_normalized_clauses FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER state_compliance_crosswalk_rows_append_only BEFORE UPDATE OR DELETE ON state_compliance_crosswalk_rows FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER regulatory_evaluations_append_only BEFORE UPDATE OR DELETE ON regulatory_evaluations FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER regulatory_evaluation_partitions_append_only BEFORE UPDATE OR DELETE ON regulatory_evaluation_partitions FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER regulatory_evaluation_partition_rows_append_only BEFORE UPDATE OR DELETE ON regulatory_evaluation_partition_rows FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER regulatory_generation_runs_append_only BEFORE UPDATE OR DELETE ON regulatory_generation_runs FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER regulatory_generation_run_source_snapshots_append_only BEFORE UPDATE OR DELETE ON regulatory_generation_run_source_snapshots FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER template_draft_versions_generated_immutable BEFORE UPDATE OR DELETE ON template_draft_versions FOR EACH ROW EXECUTE FUNCTION governed_generated_candidate_immutable_guard();
CREATE TRIGGER candidate_required_owner_assignments_append_only BEFORE UPDATE OR DELETE ON candidate_required_owner_assignments FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER department_review_decisions_append_only BEFORE UPDATE OR DELETE ON department_review_decisions FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER checklist_publication_decisions_append_only BEFORE UPDATE OR DELETE ON checklist_publication_decisions FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER regulatory_normalized_clauses_source_hash_guard BEFORE INSERT ON regulatory_normalized_clauses FOR EACH ROW EXECUTE FUNCTION validate_governed_clause_source_hash();
CREATE TRIGGER state_compliance_crosswalk_rows_source_guard BEFORE INSERT ON state_compliance_crosswalk_rows FOR EACH ROW EXECUTE FUNCTION validate_governed_crosswalk_row();
CREATE TRIGGER regulatory_evaluation_partition_rows_identity_guard BEFORE INSERT ON regulatory_evaluation_partition_rows FOR EACH ROW EXECUTE FUNCTION validate_governed_partition_identity();
CREATE TRIGGER regulatory_generation_run_source_snapshots_guard BEFORE INSERT ON regulatory_generation_run_source_snapshots FOR EACH ROW EXECUTE FUNCTION validate_governed_generation_snapshot();
CREATE TRIGGER department_review_decisions_actor_guard BEFORE INSERT ON department_review_decisions FOR EACH ROW EXECUTE FUNCTION validate_governed_decision_actor();
CREATE TRIGGER checklist_publication_decisions_actor_guard BEFORE INSERT ON checklist_publication_decisions FOR EACH ROW EXECUTE FUNCTION validate_governed_decision_actor();
CREATE TRIGGER checklist_publication_decisions_approval_guard BEFORE INSERT ON checklist_publication_decisions FOR EACH ROW EXECUTE FUNCTION validate_governed_publication_approval();
CREATE TRIGGER checklist_template_versions_governed_publication_guard BEFORE INSERT ON checklist_template_versions FOR EACH ROW EXECUTE FUNCTION validate_governed_published_template();

-- The supplied CC working copy is a secondary comparator. These five tracked
-- locators are metadata-only stable identities; neither the DOCX nor its text
-- is copied into the repository or database migration.
INSERT INTO regulatory_source_versions (
    id, source_identity, version_identity, title, source_class, source_status,
    source_locator, source_hash, effective_from, source_metadata
) VALUES (
    'NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28',
    'NCAA-CC-ANNEX6-PARTI-A610', 'SUPPLIED-2026-07-28',
    'Annex 6 Part I to NAMCAR compliance crosswalk',
    'STATE_COMPLIANCE_CROSSWALK', 'SUPPLIED_WORKING_COPY',
    'CC.zip/CC/NAMB/Annex_NAMB_A610.docx',
    'sha256:13fe82d1767320443f91ed61cf7d3b4bba0ea24f217fad45bbd9cae5fc682af2',
    '2026-07-28',
    '{"archiveHash":"sha256:3393f2313aca41ba64732fcb21913098937679b3eb4f98f815df244266a9a113","documentBytes":267905,"rowIdentityScheme":"annex-section-locator-v1"}'
);
INSERT INTO regulatory_normalized_clauses (id, regulatory_source_version_id, clause_identity, annex_identity, section_identity, clause_locator, source_hash, normalized_digest) VALUES
    ('NCAA-CC-A610-4.2.2.2', 'NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28', 'ANNEX6-4.2.2.2', 'ANNEX_6_PART_I', '4.2.2.2', 'Annex 6 Part I 4.2.2.2', 'sha256:13fe82d1767320443f91ed61cf7d3b4bba0ea24f217fad45bbd9cae5fc682af2', 'sha256:ca3b5db1ac0b98dbcf26b5004a9be3c37de6c3e514726ac78b69ba07cd043b4b'),
    ('NCAA-CC-A610-4.2.12.1', 'NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28', 'ANNEX6-4.2.12.1', 'ANNEX_6_PART_I', '4.2.12.1', 'Annex 6 Part I 4.2.12.1', 'sha256:13fe82d1767320443f91ed61cf7d3b4bba0ea24f217fad45bbd9cae5fc682af2', 'sha256:e944d44ce0e7b3fe9623675f6703b5ea1acba7f853492fbb785c72adaa835d7f'),
    ('NCAA-CC-A610-6.2.2', 'NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28', 'ANNEX6-6.2.2', 'ANNEX_6_PART_I', '6.2.2', 'Annex 6 Part I 6.2.2', 'sha256:13fe82d1767320443f91ed61cf7d3b4bba0ea24f217fad45bbd9cae5fc682af2', 'sha256:9f18a535aa5a75954ff971c1370a519be009433244f81234daba7a7c2c1bc049'),
    ('NCAA-CC-A610-6.16', 'NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28', 'ANNEX6-6.16', 'ANNEX_6_PART_I', '6.16', 'Annex 6 Part I 6.16', 'sha256:13fe82d1767320443f91ed61cf7d3b4bba0ea24f217fad45bbd9cae5fc682af2', 'sha256:3819c0a07b31c264d2b5d1ad3bd0ea8e2fee891ecded80635b3418e5b5dc874b'),
    ('NCAA-CC-A610-8.1.1', 'NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28', 'ANNEX6-8.1.1', 'ANNEX_6_PART_I', '8.1.1', 'Annex 6 Part I 8.1.1', 'sha256:13fe82d1767320443f91ed61cf7d3b4bba0ea24f217fad45bbd9cae5fc682af2', 'sha256:357ae836164d3acaee361a0e3702cb67a614a057bdd3cdeb58c16cb143808dfb');
INSERT INTO state_compliance_crosswalk_rows (id, regulatory_source_version_id, normalized_clause_id, stable_row_identity, annex_identity, section_identity, row_digest) VALUES
    ('NCAA-CC-A610-ROW-4.2.2.2', 'NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28', 'NCAA-CC-A610-4.2.2.2', 'CC:NAMB:ANNEX6:4.2.2.2', 'ANNEX_6_PART_I', '4.2.2.2', 'sha256:ca3b5db1ac0b98dbcf26b5004a9be3c37de6c3e514726ac78b69ba07cd043b4b'),
    ('NCAA-CC-A610-ROW-4.2.12.1', 'NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28', 'NCAA-CC-A610-4.2.12.1', 'CC:NAMB:ANNEX6:4.2.12.1', 'ANNEX_6_PART_I', '4.2.12.1', 'sha256:e944d44ce0e7b3fe9623675f6703b5ea1acba7f853492fbb785c72adaa835d7f'),
    ('NCAA-CC-A610-ROW-6.2.2', 'NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28', 'NCAA-CC-A610-6.2.2', 'CC:NAMB:ANNEX6:6.2.2', 'ANNEX_6_PART_I', '6.2.2', 'sha256:9f18a535aa5a75954ff971c1370a519be009433244f81234daba7a7c2c1bc049'),
    ('NCAA-CC-A610-ROW-6.16', 'NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28', 'NCAA-CC-A610-6.16', 'CC:NAMB:ANNEX6:6.16', 'ANNEX_6_PART_I', '6.16', 'sha256:3819c0a07b31c264d2b5d1ad3bd0ea8e2fee891ecded80635b3418e5b5dc874b'),
    ('NCAA-CC-A610-ROW-8.1.1', 'NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28', 'NCAA-CC-A610-8.1.1', 'CC:NAMB:ANNEX6:8.1.1', 'ANNEX_6_PART_I', '8.1.1', 'sha256:357ae836164d3acaee361a0e3702cb67a614a057bdd3cdeb58c16cb143808dfb');

-- Fix round 1: a run pins effective scope facts, not merely catalog labels.
CREATE TABLE regulatory_generation_run_scope_facts (
    generation_run_id text NOT NULL REFERENCES regulatory_generation_runs(id),
    organization_service_provider_scope_id text NOT NULL REFERENCES organization_service_provider_scopes(id),
    scope_root_id text NOT NULL REFERENCES organization_service_provider_scopes(id),
    organization_id text NOT NULL REFERENCES organizations(id),
    service_provider_type_id text NOT NULL REFERENCES service_provider_types(id),
    authorization_identifier text NOT NULL CHECK (btrim(authorization_identifier) <> ''),
    scope_status text NOT NULL CHECK (scope_status = 'ACTIVE'),
    effective_from date NOT NULL,
    effective_to date,
    regulated_target_id text NOT NULL REFERENCES regulated_targets(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (generation_run_id, organization_service_provider_scope_id),
    CHECK (effective_to IS NULL OR effective_to > effective_from)
);
CREATE INDEX regulatory_generation_run_scope_facts_lookup_idx ON regulatory_generation_run_scope_facts (generation_run_id, organization_id, service_provider_type_id, regulated_target_id);
CREATE OR REPLACE FUNCTION validate_governed_generation_scope_fact() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE scope record; run_target text; target_org text;
BEGIN
    SELECT * INTO scope FROM organization_service_provider_scopes WHERE id = NEW.organization_service_provider_scope_id;
    SELECT target_id INTO run_target FROM regulatory_generation_runs WHERE id = NEW.generation_run_id;
    SELECT COALESCE(organization_id, owner_organization_id) INTO target_org FROM regulated_targets WHERE id = NEW.regulated_target_id;
    IF scope.id IS NULL OR scope.root_id <> NEW.scope_root_id OR scope.organization_id <> NEW.organization_id OR scope.service_provider_type_id <> NEW.service_provider_type_id OR scope.authorization_identifier <> NEW.authorization_identifier OR scope.status <> NEW.scope_status OR scope.effective_from <> NEW.effective_from OR scope.effective_to IS DISTINCT FROM NEW.effective_to OR run_target <> NEW.regulated_target_id OR target_org IS DISTINCT FROM scope.organization_id OR EXISTS (SELECT 1 FROM organization_service_provider_scopes successor WHERE successor.supersedes_id = scope.id) OR scope.status <> 'ACTIVE' OR scope.effective_from > CURRENT_DATE OR (scope.effective_to IS NOT NULL AND scope.effective_to <= CURRENT_DATE) OR NOT (scope.primary_target_id = NEW.regulated_target_id OR EXISTS (SELECT 1 FROM organization_service_provider_scope_targets linked WHERE linked.organization_service_provider_scope_id = scope.id AND linked.regulated_target_id = NEW.regulated_target_id)) THEN
        RAISE EXCEPTION 'generation run scope fact must pin one current active exact scope and compatible target';
    END IF;
    RETURN NEW;
END;
$$;

ALTER TABLE template_draft_versions ADD COLUMN candidate_root_id text REFERENCES template_draft_versions(id);
ALTER TABLE template_draft_versions ADD COLUMN supersedes_candidate_id text UNIQUE REFERENCES template_draft_versions(id);
ALTER TABLE template_draft_versions ADD CONSTRAINT template_draft_versions_candidate_root_check CHECK ((generation_run_id IS NULL AND candidate_root_id IS NULL AND supersedes_candidate_id IS NULL) OR (generation_run_id IS NOT NULL AND candidate_root_id IS NOT NULL));
CREATE OR REPLACE FUNCTION validate_governed_generated_candidate() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE parent record;
BEGIN
    IF NEW.generation_run_id IS NULL THEN RETURN NEW; END IF;
    IF cardinality(NEW.question_version_ids) = 0 OR array_position(NEW.question_version_ids, '') IS NOT NULL OR EXISTS (SELECT 1 FROM unnest(NEW.question_version_ids) question_version_id WHERE NOT EXISTS (SELECT 1 FROM question_versions question WHERE question.id = question_version_id)) THEN
        RAISE EXCEPTION 'generated candidate requires nonempty immutable question-version identities';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM regulatory_generation_runs run WHERE run.id = NEW.generation_run_id AND run.status = 'GENERATED' AND run.output_digest = NEW.candidate_content_digest AND run.output_artifact IS NOT NULL) OR NOT EXISTS (SELECT 1 FROM regulatory_generation_run_scope_facts WHERE generation_run_id = NEW.generation_run_id) OR NOT EXISTS (SELECT 1 FROM regulatory_generation_run_source_snapshots WHERE generation_run_id = NEW.generation_run_id) OR (EXISTS (SELECT 1 FROM regulatory_generation_runs run WHERE run.id = NEW.generation_run_id AND run.input_artifact ? 'secondaryCrosswalkPartition') AND NOT EXISTS (SELECT 1 FROM regulatory_generation_run_crosswalk_partition_rows WHERE generation_run_id = NEW.generation_run_id)) THEN
        RAISE EXCEPTION 'generated candidate requires complete exact generation lineage and matching output digest';
    END IF;
    IF NEW.supersedes_candidate_id IS NULL THEN
        IF NEW.candidate_root_id <> NEW.id THEN RAISE EXCEPTION 'generated candidate root must equal its initial revision identity'; END IF;
    ELSE
        SELECT * INTO parent FROM template_draft_versions WHERE id = NEW.supersedes_candidate_id;
        IF parent.id IS NULL OR parent.generation_run_id IS NULL OR parent.template_id <> NEW.template_id OR parent.candidate_root_id <> NEW.candidate_root_id OR NEW.version <= parent.version OR NEW.revision <= parent.revision THEN
            RAISE EXCEPTION 'generated candidate successor must form one increasing immutable revision chain';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TABLE regulatory_generation_run_crosswalk_partition_rows (
    generation_run_id text NOT NULL REFERENCES regulatory_generation_runs(id),
    evaluation_partition_id text NOT NULL REFERENCES regulatory_evaluation_partitions(id),
    state_compliance_crosswalk_row_id text NOT NULL REFERENCES state_compliance_crosswalk_rows(id),
    stable_row_identity text NOT NULL CHECK (btrim(stable_row_identity) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (generation_run_id, state_compliance_crosswalk_row_id)
);
CREATE OR REPLACE FUNCTION validate_governed_generation_crosswalk_partition() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM regulatory_evaluation_partition_rows row JOIN regulatory_evaluation_partitions partition ON partition.id=row.partition_id WHERE row.partition_id=NEW.evaluation_partition_id AND row.state_compliance_crosswalk_row_id=NEW.state_compliance_crosswalk_row_id AND row.stable_row_identity=NEW.stable_row_identity AND partition.partition_kind='GENERATION_INPUT') THEN
        RAISE EXCEPTION 'generation run crosswalk lineage requires one exact generation-input partition row';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER regulatory_generation_run_crosswalk_partition_rows_append_only BEFORE UPDATE OR DELETE ON regulatory_generation_run_crosswalk_partition_rows FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER regulatory_generation_run_crosswalk_partition_rows_guard BEFORE INSERT ON regulatory_generation_run_crosswalk_partition_rows FOR EACH ROW EXECUTE FUNCTION validate_governed_generation_crosswalk_partition();
CREATE TABLE regulatory_generated_mapping_snapshots (
    candidate_draft_version_id text NOT NULL REFERENCES template_draft_versions(id),
    mapping_id text NOT NULL CHECK (btrim(mapping_id) <> ''),
    mapping_ordinal integer NOT NULL CONSTRAINT regulatory_generated_mapping_snapshots_mapping_ordinal_check CHECK (mapping_ordinal >= 0),
    snapshot jsonb NOT NULL CHECK (jsonb_typeof(snapshot) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (candidate_draft_version_id, mapping_id),
    CONSTRAINT regulatory_generated_mapping_snapshots_candidate_ordinal_unique UNIQUE (candidate_draft_version_id, mapping_ordinal)
);
CREATE TABLE regulatory_generated_question_snapshots (
    candidate_draft_version_id text NOT NULL REFERENCES template_draft_versions(id),
    question_id text NOT NULL CHECK (btrim(question_id) <> ''),
    snapshot jsonb NOT NULL CHECK (jsonb_typeof(snapshot) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (candidate_draft_version_id, question_id)
);
CREATE TRIGGER regulatory_generated_mapping_snapshots_append_only BEFORE UPDATE OR DELETE ON regulatory_generated_mapping_snapshots FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER regulatory_generated_question_snapshots_append_only BEFORE UPDATE OR DELETE ON regulatory_generated_question_snapshots FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE OR REPLACE FUNCTION validate_governed_decision_actor() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE membership record; effective_membership record; department_status text; unit_status text;
BEGIN
    SELECT * INTO membership FROM caa_department_memberships WHERE id = NEW.actor_department_membership_id;
    SELECT * INTO effective_membership FROM caa_department_memberships fact WHERE fact.root_id = membership.root_id AND fact.effective_from <= NEW.decided_at::date ORDER BY fact.effective_from DESC, fact.id DESC LIMIT 1;
    SELECT status INTO department_status FROM caa_department_status_facts fact WHERE fact.department_id = NEW.actor_department_id AND fact.effective_from <= NEW.decided_at::date ORDER BY fact.effective_from DESC, fact.id DESC LIMIT 1;
    SELECT status INTO unit_status FROM caa_organizational_unit_status_facts fact WHERE fact.organizational_unit_id = NEW.actor_organizational_unit_id AND fact.effective_from <= NEW.decided_at::date ORDER BY fact.effective_from DESC, fact.id DESC LIMIT 1;
    IF membership.id IS NULL OR effective_membership.id IS DISTINCT FROM membership.id OR membership.subject_id IS DISTINCT FROM NEW.actor_subject_id OR membership.department_id IS DISTINCT FROM NEW.actor_department_id OR membership.organizational_unit_id IS DISTINCT FROM NEW.actor_organizational_unit_id OR membership.membership_role IS DISTINCT FROM 'DEPARTMENT_MANAGER' OR membership.status IS DISTINCT FROM 'ACTIVE' OR department_status IS DISTINCT FROM 'ACTIVE' OR unit_status IS DISTINCT FROM 'ACTIVE' OR membership.effective_from > NEW.decided_at::date OR (membership.effective_to IS NOT NULL AND membership.effective_to <= NEW.decided_at::date) THEN
        RAISE EXCEPTION 'decision actor has no current matching Department Manager assignment';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER regulatory_generation_run_scope_facts_append_only BEFORE UPDATE OR DELETE ON regulatory_generation_run_scope_facts FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TRIGGER regulatory_generation_run_scope_facts_guard BEFORE INSERT ON regulatory_generation_run_scope_facts FOR EACH ROW EXECUTE FUNCTION validate_governed_generation_scope_fact();
CREATE TRIGGER template_draft_versions_generated_lineage_guard BEFORE INSERT ON template_draft_versions FOR EACH ROW EXECUTE FUNCTION validate_governed_generated_candidate();

-- Task 5 Admin boundary: commands retain their audit/idempotency identity and
-- successor candidates remain immutable. The sole mutable transition is the
-- exact Generated Draft submission to Department Review.
CREATE TABLE regulatory_source_gap_facts (
    id text PRIMARY KEY,
    regulatory_source_version_id text NOT NULL REFERENCES regulatory_source_versions(id),
    gap_id text NOT NULL CHECK (btrim(gap_id) <> ''),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    ordinal integer NOT NULL CHECK (ordinal >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (regulatory_source_version_id, gap_id)
);
CREATE INDEX regulatory_source_gap_facts_source_idx ON regulatory_source_gap_facts (regulatory_source_version_id, ordinal, gap_id);
CREATE TRIGGER regulatory_source_gap_facts_append_only BEFORE UPDATE OR DELETE ON regulatory_source_gap_facts FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE TABLE governed_candidate_commands (
    id text PRIMARY KEY,
    command_kind text NOT NULL CHECK (command_kind IN ('IMPORTED_GENERATION_RUN', 'FAILED_IMPORT', 'REVISION_CREATED', 'DEPARTMENT_REVIEW_SUBMITTED')),
    operation_id text NOT NULL UNIQUE CHECK (btrim(operation_id) <> ''),
    idempotency_key text NOT NULL UNIQUE CHECK (btrim(idempotency_key) <> ''),
    semantic_payload_digest text NOT NULL CHECK (governed_sha256(semantic_payload_digest)),
    generation_run_id text NOT NULL REFERENCES regulatory_generation_runs(id),
    candidate_draft_version_id text REFERENCES template_draft_versions(id),
    candidate_revision bigint,
    candidate_content_digest text CHECK (candidate_content_digest IS NULL OR governed_sha256(candidate_content_digest)),
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    audit_event_id text NOT NULL REFERENCES audit_events(event_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (
        (command_kind = 'FAILED_IMPORT' AND candidate_draft_version_id IS NULL AND candidate_revision IS NULL AND candidate_content_digest IS NULL)
        OR
        (command_kind <> 'FAILED_IMPORT' AND candidate_draft_version_id IS NOT NULL AND candidate_revision IS NOT NULL AND candidate_content_digest IS NOT NULL)
    ),
    FOREIGN KEY (candidate_draft_version_id, candidate_revision, candidate_content_digest)
        REFERENCES template_draft_versions(id, revision, candidate_content_digest)
);
CREATE INDEX governed_candidate_commands_run_candidate_idx ON governed_candidate_commands (generation_run_id, candidate_draft_version_id, candidate_revision);
CREATE TRIGGER governed_candidate_commands_append_only BEFORE UPDATE OR DELETE ON governed_candidate_commands FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
CREATE OR REPLACE FUNCTION governed_generated_candidate_immutable_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.generation_run_id IS NULL THEN RETURN NEW; END IF;
    IF (OLD.status, NEW.status) IN (
           ('GENERATED_DRAFT', 'DEPARTMENT_REVIEW'),
           ('DEPARTMENT_REVIEW', 'RETURNED'),
           ('DEPARTMENT_REVIEW', 'REJECTED'),
           ('DEPARTMENT_REVIEW', 'TECHNICALLY_APPROVED'),
           ('TECHNICALLY_APPROVED', 'PUBLISHED')
       )
       AND NEW.id = OLD.id AND NEW.template_id = OLD.template_id AND NEW.version = OLD.version
       AND NEW.owner_role = OLD.owner_role AND NEW.creator_subject_id = OLD.creator_subject_id
       AND NEW.change_reason = OLD.change_reason AND NEW.question_version_ids = OLD.question_version_ids
       AND NEW.revision = OLD.revision AND NEW.generation_run_id = OLD.generation_run_id
       AND NEW.candidate_content_digest = OLD.candidate_content_digest
       AND NEW.candidate_schema_version = OLD.candidate_schema_version
       AND NEW.candidate_root_id = OLD.candidate_root_id
       AND NEW.supersedes_candidate_id IS NOT DISTINCT FROM OLD.supersedes_candidate_id
    THEN RETURN NEW; END IF;
    RAISE EXCEPTION 'generated candidate revisions are immutable except governed status transitions';
END;
$$;
CREATE OR REPLACE FUNCTION validate_governed_generated_candidate() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE parent record;
BEGIN
    IF NEW.generation_run_id IS NULL THEN RETURN NEW; END IF;
    IF cardinality(NEW.question_version_ids) = 0 OR array_position(NEW.question_version_ids, '') IS NOT NULL OR EXISTS (SELECT 1 FROM unnest(NEW.question_version_ids) question_version_id WHERE NOT EXISTS (SELECT 1 FROM question_versions question WHERE question.id = question_version_id)) THEN RAISE EXCEPTION 'generated candidate requires nonempty immutable question-version identities'; END IF;
    IF NOT EXISTS (SELECT 1 FROM regulatory_generation_runs run WHERE run.id = NEW.generation_run_id AND run.status = 'GENERATED' AND run.output_artifact IS NOT NULL) OR NOT EXISTS (SELECT 1 FROM regulatory_generation_run_scope_facts WHERE generation_run_id = NEW.generation_run_id) OR NOT EXISTS (SELECT 1 FROM regulatory_generation_run_source_snapshots WHERE generation_run_id = NEW.generation_run_id) THEN RAISE EXCEPTION 'generated candidate requires complete exact generation lineage'; END IF;
    IF NEW.supersedes_candidate_id IS NULL THEN
        IF NEW.candidate_root_id <> NEW.id OR NOT EXISTS (SELECT 1 FROM regulatory_generation_runs run WHERE run.id = NEW.generation_run_id AND run.output_digest = NEW.candidate_content_digest) THEN RAISE EXCEPTION 'generated candidate root must pin its exact generated output digest'; END IF;
    ELSE
        SELECT * INTO parent FROM template_draft_versions WHERE id = NEW.supersedes_candidate_id;
        IF parent.id IS NULL OR parent.generation_run_id IS NULL OR parent.template_id <> NEW.template_id OR parent.candidate_root_id <> NEW.candidate_root_id OR parent.generation_run_id <> NEW.generation_run_id OR NEW.version <= parent.version OR NEW.revision <= parent.revision THEN RAISE EXCEPTION 'generated candidate successor must form one increasing immutable revision chain'; END IF;
    END IF;
    RETURN NEW;
END;
$$;
