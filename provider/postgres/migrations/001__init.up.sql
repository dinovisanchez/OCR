CREATE TABLE IF NOT EXISTS extractions (
    id SERIAL PRIMARY KEY,
    document_type Text NOT NULL,
    document_identification bigint NOT NULL,
    request Text NOT NULL,
    response Text NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
); 