INSERT INTO permissions (id, name, description, resource, action, created_at)
VALUES
    (uuid_generate_v4(), 'land:read', 'Read land acquisitions', 'land', 'read', NOW()),
    (uuid_generate_v4(), 'land:create', 'Create land acquisitions', 'land', 'create', NOW()),
    (uuid_generate_v4(), 'land:update', 'Update land acquisitions', 'land', 'update', NOW())
ON CONFLICT (name) DO UPDATE SET
    description = EXCLUDED.description,
    resource = EXCLUDED.resource,
    action = EXCLUDED.action;

INSERT INTO roles (id, name, description, created_at, updated_at)
VALUES (uuid_generate_v4(), 'senior_specialist', 'Ахлах мэргэжилтэн', NOW(), NOW())
ON CONFLICT (name) DO UPDATE SET
    description = EXCLUDED.description,
    updated_at = NOW();

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.name IN ('land:read', 'land:create', 'land:update')
WHERE r.name = 'senior_specialist'
ON CONFLICT DO NOTHING;
