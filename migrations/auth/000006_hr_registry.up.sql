CREATE TABLE IF NOT EXISTS organization (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    parent_id UUID NULL REFERENCES organization(id),
    name TEXT NOT NULL,
    short_name TEXT,
    register_no TEXT UNIQUE,
    address TEXT,
    phone TEXT,
    email TEXT,
    external_id TEXT,
    source TEXT NOT NULL DEFAULT 'local',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_organization_parent_id ON organization(parent_id);
CREATE INDEX IF NOT EXISTS idx_organization_deleted_at ON organization(deleted_at);

CREATE TABLE IF NOT EXISTS department (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organization(id),
    parent_id UUID NULL REFERENCES department(id),
    code TEXT,
    name TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_department_org_name_active ON department(organization_id, name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_department_organization_id ON department(organization_id);
CREATE INDEX IF NOT EXISTS idx_department_parent_id ON department(parent_id);

CREATE TABLE IF NOT EXISTS position (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code TEXT,
    name TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_position_name_active ON position(name) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS person (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    person_type TEXT NOT NULL CHECK (person_type IN ('citizen','legal')),
    register_no TEXT NOT NULL,
    last_name TEXT,
    first_name TEXT,
    legal_name TEXT,
    civil_id TEXT,
    birth_date DATE,
    gender TEXT CHECK (gender IN ('M','F') OR gender IS NULL),
    phone TEXT,
    email TEXT,
    address TEXT,
    external_id TEXT,
    source TEXT NOT NULL DEFAULT 'local',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    created_by TEXT,
    updated_by TEXT,
    CHECK (
        (person_type = 'citizen' AND COALESCE(last_name, '') <> '' AND COALESCE(first_name, '') <> '')
        OR
        (person_type = 'legal' AND COALESCE(legal_name, '') <> '')
    )
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_person_register_no_active ON person(register_no) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_person_search ON person(register_no, first_name, last_name, legal_name);

CREATE TABLE IF NOT EXISTS employee (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    person_id UUID NOT NULL REFERENCES person(id),
    organization_id UUID NOT NULL REFERENCES organization(id),
    department_id UUID NULL REFERENCES department(id),
    position_id UUID NOT NULL REFERENCES position(id),
    employee_no TEXT,
    work_email TEXT,
    work_phone TEXT,
    hired_at DATE,
    released_at DATE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    created_by TEXT,
    updated_by TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_employee_person_active ON employee(person_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS ux_employee_no_active ON employee(employee_no) WHERE deleted_at IS NULL AND employee_no IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_employee_organization_id ON employee(organization_id);
CREATE INDEX IF NOT EXISTS idx_employee_department_id ON employee(department_id);
CREATE INDEX IF NOT EXISTS idx_employee_position_id ON employee(position_id);

ALTER TABLE users ADD COLUMN IF NOT EXISTS employee_id UUID NULL REFERENCES employee(id);
CREATE UNIQUE INDEX IF NOT EXISTS ux_users_employee_active ON users(employee_id) WHERE employee_id IS NOT NULL AND deleted_at IS NULL;
