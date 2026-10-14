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
    es.enrollment_id = $1;

-- name: RemoveAllEnrollmentSets :execresult
DELETE FROM
    enrollment_sets
WHERE
    enrollment_id = $1;

-- name: GetDeclaration :one
SELECT
    d.identifier,
    d.type,
    d.payload,
    d.server_token,
    jsonb_build_object(
        'Identifier',  d.identifier,
        'Type',        d.type,
        'Payload',     d.payload,
        'ServerToken', d.server_token
    ) AS declaration
FROM
    declarations d
WHERE
    d.identifier = $1;

-- name: GetDDMDeclaration :one
SELECT
    jsonb_build_object(
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
    d.identifier = sqlc.arg(identifier) AND
    es.enrollment_id = sqlc.arg(enrollment_id) AND
    d.type LIKE sqlc.arg(type)
LIMIT 1;

-- name: RemoveDeclarationStatus :exec
DELETE FROM
    status_declarations
WHERE
    enrollment_id = $1;

-- name: PutDeclarationStatus :exec
INSERT INTO status_declarations (
    enrollment_id,
    item_type,
    declaration_identifier,
    active,
    valid,
    server_token,
    reasons,
    status_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(enrollment_id),
    sqlc.arg(item_type),
    sqlc.arg(declaration_identifier),
    sqlc.arg(active),
    sqlc.arg(valid),
    sqlc.arg(server_token),
    sqlc.narg(reasons),
    sqlc.narg(status_id),
    sqlc.arg(now)::bigint,
    sqlc.arg(now)::bigint
);

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
WHERE
    sd.enrollment_id = ANY(sqlc.arg(ids)::text[])
ORDER BY
    sd.enrollment_id;

-- name: InsertStatusError :exec
INSERT INTO status_errors (
    enrollment_id,
    path,
    error,
    status_id,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(enrollment_id),
    sqlc.arg(path),
    sqlc.arg(error),
    sqlc.narg(status_id),
    sqlc.arg(now)::bigint,
    sqlc.arg(now)::bigint
);

-- Keeps only the newest (offset) errors for the enrollment.
-- name: DeleteStatusErrors :exec
DELETE FROM
    status_errors
WHERE
    status_errors.enrollment_id = sqlc.arg(enrollment_id)
    AND status_errors.id <= (
        SELECT se.id FROM status_errors se
        WHERE se.enrollment_id = sqlc.arg(enrollment_id)
        ORDER BY se.id DESC
        LIMIT 1 OFFSET sqlc.arg(keep)
    );

-- name: SelectStatusErrors :many
SELECT
    enrollment_id,
    path,
    error,
    status_id,
    created_at
FROM
    status_errors
WHERE
    enrollment_id = ANY(sqlc.arg(ids)::text[])
ORDER BY
    enrollment_id, id
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: InsertStatusReport :exec
INSERT INTO status_reports (
    enrollment_id,
    status_id,
    status_report,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(enrollment_id),
    sqlc.narg(status_id),
    sqlc.arg(status_report),
    sqlc.arg(now)::bigint,
    sqlc.arg(now)::bigint
);

-- Keeps only the newest (offset) status reports for the enrollment.
-- name: DeleteStatusReports :exec
DELETE FROM
    status_reports
WHERE
    status_reports.enrollment_id = sqlc.arg(enrollment_id)
    AND status_reports.id <= (
        SELECT sr.id FROM status_reports sr
        WHERE sr.enrollment_id = sqlc.arg(enrollment_id)
        ORDER BY sr.id DESC
        LIMIT 1 OFFSET sqlc.arg(keep)
    );

-- Index 0 is the most recent status report for the enrollment.
-- name: SelectStatusReportByIndex :one
SELECT
    status_id,
    created_at,
    status_report
FROM
    status_reports
WHERE
    enrollment_id = sqlc.arg(enrollment_id)
ORDER BY
    id DESC
LIMIT 1 OFFSET sqlc.arg(row_offset);

-- name: SelectStatusReportByStatusID :one
SELECT
    sr.status_id,
    sr.created_at,
    sr.status_report,
    (
        SELECT COUNT(*) FROM status_reports
        WHERE status_reports.enrollment_id = sr.enrollment_id
            AND status_reports.id > sr.id
    ) AS idx
FROM
    status_reports sr
WHERE
    sr.enrollment_id = $1
    AND sr.status_id = $2
ORDER BY
    sr.id DESC
LIMIT 1;

-- The WHERE on the conflict update leaves an unchanged declaration untouched,
-- so it reports no affected rows and keeps its server token.
-- name: StoreDeclaration :execresult
INSERT INTO declarations
    (identifier, type, payload, server_token, created_at, updated_at)
VALUES
    (
        sqlc.arg(identifier)::text,
        sqlc.arg(type)::text,
        sqlc.arg(payload)::jsonb,
        encode(sha256(convert_to(concat(sqlc.arg(identifier)::text, sqlc.arg(type)::text, sqlc.arg(payload)::jsonb::text, sqlc.arg(now)::bigint::text, '0'), 'UTF8')), 'hex'),
        sqlc.arg(now)::bigint,
        sqlc.arg(now)::bigint
    )
ON CONFLICT (identifier) DO UPDATE
SET
    type         = excluded.type,
    payload      = excluded.payload,
    server_token = encode(sha256(convert_to(concat(excluded.identifier, excluded.type, excluded.payload::text, declarations.created_at::text, declarations.touched_ct::text), 'UTF8')), 'hex'),
    updated_at   = excluded.updated_at
WHERE
    declarations.type IS DISTINCT FROM excluded.type OR
    declarations.payload IS DISTINCT FROM excluded.payload;

-- name: TouchDeclaration :execresult
UPDATE
    declarations
SET
    touched_ct   = touched_ct + 1,
    server_token = encode(sha256(convert_to(concat(identifier, type, payload::text, created_at::text, (touched_ct + 1)::text), 'UTF8')), 'hex'),
    updated_at   = sqlc.arg(now)::bigint
WHERE
    identifier = sqlc.arg(identifier);

-- name: DeleteDeclaration :execresult
DELETE FROM declarations WHERE identifier = $1;

-- name: GetDeclarationModTime :one
SELECT updated_at FROM declarations WHERE identifier = $1;

-- name: GetDeclarationSets :many
SELECT set_name FROM set_declarations WHERE declaration_identifier = $1;

-- name: GetDeclarationIdentifiers :many
SELECT identifier FROM declarations;

-- name: GetSetDeclarations :many
SELECT declaration_identifier FROM set_declarations WHERE set_name = $1;

-- name: StoreSetDeclaration :execresult
INSERT INTO set_declarations
    (declaration_identifier, set_name, created_at, updated_at)
VALUES
    (sqlc.arg(declaration_identifier), sqlc.arg(set_name), sqlc.arg(now)::bigint, sqlc.arg(now)::bigint)
ON CONFLICT DO NOTHING;

-- name: RemoveSetDeclaration :execresult
DELETE FROM set_declarations
WHERE
    set_name = $1 AND
    declaration_identifier = $2;

-- name: GetSets :many
SELECT DISTINCT set_name FROM set_declarations;

-- name: GetEnrollmentSets :many
SELECT set_name FROM enrollment_sets WHERE enrollment_id = $1;

-- name: StoreEnrollmentSet :execresult
INSERT INTO enrollment_sets
    (enrollment_id, set_name, created_at, updated_at)
VALUES
    (sqlc.arg(enrollment_id), sqlc.arg(set_name), sqlc.arg(now)::bigint, sqlc.arg(now)::bigint)
ON CONFLICT DO NOTHING;

-- name: RemoveEnrollmentSet :execresult
DELETE FROM enrollment_sets
WHERE
    enrollment_id = $1 AND
    set_name = $2;

-- An empty array matches nothing, so unused filters drop out of the OR.
-- name: GetEnrollmentIDs :many
SELECT DISTINCT
    es.enrollment_id
FROM
    enrollment_sets es
    LEFT JOIN set_declarations sd
        ON sd.set_name = es.set_name
    LEFT JOIN declarations d
        ON d.identifier = sd.declaration_identifier
WHERE
    d.identifier = ANY(sqlc.arg(declarations)::text[]) OR
    es.set_name = ANY(sqlc.arg(sets)::text[]) OR
    es.enrollment_id = ANY(sqlc.arg(ids)::text[]);

-- DISTINCT collapses values a report repeats: PostgreSQL refuses to update
-- the same row twice in one INSERT ... ON CONFLICT DO UPDATE.
-- name: PutStatusValues :exec
INSERT INTO status_values
    (enrollment_id, path, container_type, value_type, value, status_id, created_at, updated_at)
SELECT DISTINCT
    sqlc.arg(enrollment_id)::text,
    v.path,
    v.container_type,
    v.value_type,
    v.value,
    sqlc.narg(status_id)::text,
    sqlc.arg(now)::bigint,
    sqlc.arg(now)::bigint
FROM
    -- the unnests zip the equal-length arrays by position.
    (
        SELECT
            unnest(sqlc.arg(paths)::text[])           AS path,
            unnest(sqlc.arg(container_types)::text[]) AS container_type,
            unnest(sqlc.arg(value_types)::text[])     AS value_type,
            unnest(sqlc.arg(vals)::text[])            AS value
    ) v
ON CONFLICT (enrollment_id, path, container_type, value_type, value) DO UPDATE
SET
    updated_at = excluded.updated_at,
    status_id  = excluded.status_id;

-- name: GetStatusValues :many
SELECT
    enrollment_id,
    path,
    value,
    status_id,
    updated_at
FROM
    status_values
WHERE
    enrollment_id = ANY(sqlc.arg(ids)::text[]) AND
    (sqlc.narg(path_prefix)::text IS NULL OR path LIKE sqlc.narg(path_prefix)::text)
ORDER BY
    enrollment_id, created_at;
