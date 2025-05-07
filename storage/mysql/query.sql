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

-- name: RemoveDeclarationStatus :exec
DELETE FROM
    status_declarations
WHERE
    enrollment_id = ?;

-- name: PutDeclarationStatus :exec
INSERT INTO status_declarations (
    enrollment_id,
    item_type,
    declaration_identifier,
    active,
    valid,
    server_token,
    reasons,
    status_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetDeclarationStatus :many
SELECT
    sd.enrollment_id,
    sd.declaration_identifier,
    sd.active,
    sd.valid,
    sd.reasons,
    sd.server_token,
    sd.updated_at,
    sd.status_id,
    sd.server_token = COALESCE(d.server_token, '') AS current
FROM
    status_declarations sd
    LEFT JOIN declarations d
        ON sd.declaration_identifier = d.identifier
    LEFT JOIN set_declarations setd
        ON d.identifier = setd.declaration_identifier
    LEFT JOIN enrollment_sets es
        ON setd.set_name = es.set_name AND sd.enrollment_id = es.enrollment_id
WHERE
    sd.enrollment_id IN (sqlc.slice('ids'))
ORDER BY
    sd.enrollment_id;
