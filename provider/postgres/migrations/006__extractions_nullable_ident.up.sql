ALTER TABLE extractions
    ALTER COLUMN document_identification DROP NOT NULL;

ALTER TABLE extractions
    ALTER COLUMN document_identification TYPE TEXT
    USING document_identification::TEXT;

ALTER TABLE document_type_prompts
    ADD COLUMN identification_number VARCHAR(255);
