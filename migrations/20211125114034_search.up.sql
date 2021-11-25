CREATE VIEW search_items AS
SELECT 
    text 'Thing' AS entity_type, text 'things' AS uri_path, id AS id, name || ' - ' || description AS label, organisation_id, ts
FROM
    things
UNION ALL
SELECT 
    text 'Organisation' AS entity_type, text 'organisations' AS uri_path, id AS id, name AS label, id AS organisation_id, ts
FROM
    organisations
;
