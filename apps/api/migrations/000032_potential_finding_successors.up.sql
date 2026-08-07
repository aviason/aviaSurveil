-- Returned Potential Findings are immutable review outcomes. A correction is
-- a new successor root for the same checklist response, never an overwrite
-- of the returned record.
ALTER TABLE potential_findings
    ADD COLUMN IF NOT EXISTS supersedes_potential_finding_id text
        REFERENCES potential_findings(id);

DROP INDEX IF EXISTS potential_findings_checklist_response_idx;

CREATE UNIQUE INDEX IF NOT EXISTS potential_findings_active_response_idx
    ON potential_findings (checklist_response_id)
    WHERE status <> 'RETURNED';

CREATE INDEX IF NOT EXISTS potential_findings_successor_idx
    ON potential_findings (supersedes_potential_finding_id)
    WHERE supersedes_potential_finding_id IS NOT NULL;
