-- Хэрэглэгчид илгээх системийн мэдэгдэл.
-- recipient_id / actor_id — auth DB-ийн users.id (өөр сан тул FK байхгүй).
CREATE TABLE IF NOT EXISTS notification (
    id UUID PRIMARY KEY,
    recipient_id UUID NOT NULL,
    actor_id UUID,
    actor_name TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    resource_type TEXT NOT NULL DEFAULT '',
    resource_id TEXT NOT NULL DEFAULT '',
    acquisition_id UUID,
    parcel_id UUID,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at TIMESTAMPTZ
);

-- Хонхны жагсаалт үргэлж (хүлээн авагч, уншаагүй эхэндээ, шинэ нь эхэндээ) гэж уншигдана.
CREATE INDEX IF NOT EXISTS idx_notification_recipient_created
    ON notification (recipient_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notification_recipient_unread
    ON notification (recipient_id) WHERE is_read = FALSE;
