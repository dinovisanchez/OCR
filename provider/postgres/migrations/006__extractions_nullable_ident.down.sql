ALTER TABLE document_type_prompts
    DROP COLUMN IF EXISTS identification_number;

ALTER TABLE extractions
    ALTER COLUMN document_identification TYPE BIGINT
    USING NULLIF(document_identification, '')::BIGINT;

ALTER TABLE extractions
    ALTER COLUMN document_identification SET NOT NULL;
