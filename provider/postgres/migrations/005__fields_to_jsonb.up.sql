-- Cambiar fields de TEXT[] a JSONB: cada elemento pasa a ser {"name": "...", "type": "string"}.
-- Los tipos específicos se ajustan luego por document_type.
--
-- Postgres no permite subqueries dentro de `ALTER COLUMN ... TYPE ... USING`,
-- así que lo hacemos por pasos: columna nueva → poblar → drop vieja → rename.

ALTER TABLE document_type_prompts
    ADD COLUMN fields_new JSONB NOT NULL DEFAULT '[]'::jsonb;

UPDATE document_type_prompts
SET fields_new = COALESCE(
    (SELECT jsonb_agg(jsonb_build_object('name', f, 'type', 'string'))
     FROM unnest(fields) AS f),
    '[]'::jsonb
);

ALTER TABLE document_type_prompts DROP COLUMN fields;
ALTER TABLE document_type_prompts RENAME COLUMN fields_new TO fields;
UPDATE document_type_prompts
SET fields = '[
    {"name":"account_number","type":"string"},
    {"name":"niu","type":"string"},
    {"name":"meter_number","type":"string"},
    {"name":"city","type":"string"},
    {"name":"address","type":"string"},
    {"name":"level","type":"string"},
    {"name":"property","type":"string"},
    {"name":"use","type":"string"},
    {"name":"estrato","type":"string"},
    {"name":"operator","type":"string"},
    {"name":"marketer","type":"string"},
    {"name":"month1","type":"integer"},
    {"name":"month2","type":"integer"},
    {"name":"month3","type":"integer"},
    {"name":"month4","type":"integer"},
    {"name":"month5","type":"integer"},
    {"name":"month6","type":"integer"},
    {"name":"cu","type":"number"},
    {"name":"period","type":"string"},
    {"name":"is_commercial","type":"boolean"}
]'::jsonb
WHERE document_type = 'energy_invoice';
