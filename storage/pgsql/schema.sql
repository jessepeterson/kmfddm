CREATE TABLE declarations (
    identifier VARCHAR(255) NOT NULL,
    type       VARCHAR(255) NOT NULL,
    payload    JSONB NOT NULL,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    touched_ct INTEGER DEFAULT 0 NOT NULL,

    -- computed in query.sql from the identifier, type, payload,
    -- created_at and touched_ct columns.
    server_token CHAR(64) NOT NULL,

    PRIMARY KEY (identifier),

    CHECK (type != '')
);

CREATE INDEX declarations_type_idx ON declarations (type);

CREATE TABLE set_declarations (
    set_name               VARCHAR(255) NOT NULL,
    declaration_identifier VARCHAR(255) NOT NULL,

    PRIMARY KEY (set_name, declaration_identifier),

    CHECK (set_name != ''),
    CHECK (declaration_identifier != ''),

    FOREIGN KEY (declaration_identifier)
        REFERENCES declarations (identifier),

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE INDEX set_declarations_declaration_identifier_idx ON set_declarations (declaration_identifier);

CREATE TABLE enrollment_sets (
    enrollment_id VARCHAR(255) NOT NULL,
    set_name      VARCHAR(255) NOT NULL,

    PRIMARY KEY (enrollment_id, set_name),

    CHECK (enrollment_id != ''),
    CHECK (set_name != ''),

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE INDEX enrollment_sets_set_name_idx ON enrollment_sets (set_name);

CREATE TABLE status_declarations (
    enrollment_id VARCHAR(255) NOT NULL,

    -- we don't setup a FK here because the reported identifier may be deleted
    -- or otherwise not tracked in our DB.
    declaration_identifier VARCHAR(255) NOT NULL,

    active       BOOLEAN NOT NULL,
    valid        VARCHAR(255) NOT NULL,
    server_token VARCHAR(255) NOT NULL,
    -- the shorter item type (e.g. "configuration"), not the full declaration
    -- type: we may get status on declarations we don't otherwise know about.
    item_type    VARCHAR(255) NOT NULL,

    reasons JSONB NULL,

    status_id VARCHAR(255) NULL,

    PRIMARY KEY (enrollment_id, declaration_identifier),

    CHECK (enrollment_id != ''),
    CHECK (declaration_identifier != ''),

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE TABLE status_values (
    enrollment_id VARCHAR(128) NOT NULL,

    path           VARCHAR(255) NOT NULL,
    container_type VARCHAR(6) NOT NULL, -- object|array
    value_type     VARCHAR(7) NOT NULL, -- string|number|boolean
    value          VARCHAR(255) NOT NULL,

    status_id VARCHAR(255) NULL,

    UNIQUE (enrollment_id, path, container_type, value_type, value),

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE INDEX status_values_path_idx ON status_values (path);

CREATE TABLE status_errors (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    enrollment_id VARCHAR(255) NOT NULL,

    path  VARCHAR(255) NOT NULL,
    error JSONB NOT NULL,

    status_id VARCHAR(255) NULL,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE INDEX status_errors_created_at_idx ON status_errors (created_at);
CREATE INDEX status_errors_enrollment_id_id_idx ON status_errors (enrollment_id, id);

CREATE TABLE status_reports (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    enrollment_id VARCHAR(255) NOT NULL,

    -- json, not jsonb: keeps the report as sent, including \u0000.
    status_report JSON NOT NULL,

    status_id VARCHAR(255) NULL,

    CHECK (enrollment_id != ''),
    CHECK (status_report::text != 'null'),

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE INDEX status_reports_created_at_idx ON status_reports (created_at);
CREATE INDEX status_reports_enrollment_id_id_idx ON status_reports (enrollment_id, id);
