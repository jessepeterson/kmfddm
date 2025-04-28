-- name: GetManifestItems :many
SELECT DISTINCT
    d.identifier,
    d.type,
    d.server_token
FROM
    declarations d
    INNER JOIN set_declarations sd
        ON d.identifier = sd.declaration_identifier
    INNER JOIN enrollment_sets es
        ON sd.set_name = es.set_name
WHERE
    es.enrollment_id = ?;

-- name: RemoveAllEnrollmentSets :execresult
DELETE FROM
    enrollment_sets
WHERE
    enrollment_id = ?;

-- name: GetDeclaration :one
SELECT
    d.identifier,
    d.type,
    d.payload,
    d.server_token,
    JSON_OBJECT(
        'Identifier',  d.identifier,
        'Type',        d.type,
        'Payload',     d.payload,
        'ServerToken', d.server_token
    ) AS declaration
FROM
    declarations d
WHERE
    d.identifier = ?;

-- name: GetDeclarationReferences :many
SELECT
    declaration_reference
FROM
    declaration_references
WHERE
    declaration_identifier = ?;

-- name: GetDDMDeclaration :one
SELECT
    JSON_OBJECT(
        'Identifier',  d.identifier,
        'Type',        d.type,
        'Payload',     d.payload,
        'ServerToken', d.server_token
    ) AS declaration
FROM
    declarations d
    INNER JOIN set_declarations sd
        ON d.identifier = sd.declaration_identifier
    INNER JOIN enrollment_sets es
        ON sd.set_name = es.set_name
WHERE
    d.identifier = ? AND
    es.enrollment_id = ? AND
    d.type LIKE ?;
