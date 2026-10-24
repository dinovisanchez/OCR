-- Volver a TEXT[] extrayendo solo los nombres (pierde la información de tipo).
-- Mismo enfoque que el up: columna nueva → poblar → drop → rename, porque
-- Postgres no permite subqueries en `ALTER COLUMN ... TYPE ... USING`.

ALTER TABLE document_type_prompts
    ADD COLUMN fields_new TEXT[] NOT NULL DEFAULT '{}'::text[];

UPDATE document_type_prompts
SET fields_new = COALESCE(
    (SELECT array_agg(elem->>'name')
     FROM jsonb_array_elements(fields) AS elem),
    ARRAY[]::TEXT[]
);

ALTER TABLE document_type_prompts DROP COLUMN fields;

ALTER TABLE document_type_prompts RENAME COLUMN fields_new TO fields;
