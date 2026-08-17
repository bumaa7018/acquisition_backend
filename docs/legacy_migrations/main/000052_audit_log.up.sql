-- Ажилтнуудын бизнес үйлдлийн append-only audit log.
-- actor_id нь auth DB-ийн users.id тул FK тавихгүй.
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY,
    actor_id UUID,
    actor_name TEXT NOT NULL DEFAULT '',
    actor_position TEXT NOT NULL DEFAULT '',
    actor_roles JSONB NOT NULL DEFAULT '[]'::jsonb,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL DEFAULT '',
    acquisition_id UUID,
    parcel_id UUID,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip_address TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_created
    ON audit_log (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_actor_created
    ON audit_log (actor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_resource_created
    ON audit_log (resource_type, resource_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_acquisition_created
    ON audit_log (acquisition_id, created_at DESC) WHERE acquisition_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_audit_log_parcel_created
    ON audit_log (parcel_id, created_at DESC) WHERE parcel_id IS NOT NULL;
