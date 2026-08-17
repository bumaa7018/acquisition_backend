DELETE FROM sdplatform.sd_form_permission
WHERE form_code IN ('land', 'compensation', 'decision', 'users', 'roles', 'permissions', 'hr', 'admin', 'audit');
