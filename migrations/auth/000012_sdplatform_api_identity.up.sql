CREATE TABLE IF NOT EXISTS sdplatform.sd_user_api_meta (
  user_id int4 PRIMARY KEY REFERENCES sdplatform.sd_user(user_id) ON DELETE CASCADE ON UPDATE CASCADE,
  api_uuid uuid NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
  employee_uuid uuid NULL,
  position varchar(120) NULL,
  tokens_valid_from timestamptz NULL
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_role_api_meta (
  role_id int4 PRIMARY KEY REFERENCES sdplatform.sd_role(role_id) ON DELETE CASCADE ON UPDATE CASCADE,
  api_uuid uuid NOT NULL UNIQUE DEFAULT uuid_generate_v4()
);

CREATE TABLE IF NOT EXISTS sdplatform.sd_permission_api_meta (
  permission_id int4 PRIMARY KEY REFERENCES sdplatform.sd_permission(permission_id) ON DELETE CASCADE ON UPDATE CASCADE,
  api_uuid uuid NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
  resource varchar(100) NOT NULL DEFAULT '',
  action varchar(100) NOT NULL DEFAULT ''
);

INSERT INTO sdplatform.sd_user_api_meta (user_id, api_uuid, employee_uuid, position, tokens_valid_from)
SELECT su.user_id, u.id, u.employee_id, u.position, u.tokens_valid_from
FROM sdplatform.sd_user su
JOIN users u ON u.email = su.email
ON CONFLICT (user_id) DO UPDATE SET
  api_uuid = EXCLUDED.api_uuid,
  employee_uuid = EXCLUDED.employee_uuid,
  position = EXCLUDED.position,
  tokens_valid_from = EXCLUDED.tokens_valid_from;

INSERT INTO sdplatform.sd_user_api_meta (user_id)
SELECT su.user_id
FROM sdplatform.sd_user su
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO sdplatform.sd_role_api_meta (role_id, api_uuid)
SELECT sr.role_id, r.id
FROM sdplatform.sd_role sr
JOIN roles r ON r.name = sr.name
ON CONFLICT (role_id) DO UPDATE SET api_uuid = EXCLUDED.api_uuid;

INSERT INTO sdplatform.sd_role_api_meta (role_id)
SELECT sr.role_id
FROM sdplatform.sd_role sr
ON CONFLICT (role_id) DO NOTHING;

INSERT INTO sdplatform.sd_permission_api_meta (permission_id, api_uuid, resource, action)
SELECT sp.permission_id, p.id, p.resource, p.action
FROM sdplatform.sd_permission sp
JOIN permissions p ON p.name = sp.name
ON CONFLICT (permission_id) DO UPDATE SET
  api_uuid = EXCLUDED.api_uuid,
  resource = EXCLUDED.resource,
  action = EXCLUDED.action;

INSERT INTO sdplatform.sd_permission_api_meta (permission_id, resource, action)
SELECT sp.permission_id,
       split_part(sp.name, ':', 1),
       split_part(sp.name, ':', 2)
FROM sdplatform.sd_permission sp
ON CONFLICT (permission_id) DO NOTHING;
