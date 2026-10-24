ALTER TABLE document_type_prompts
ADD COLUMN fields TEXT[] NOT NULL DEFAULT '{}';
