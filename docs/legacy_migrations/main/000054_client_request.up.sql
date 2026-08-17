CREATE TABLE IF NOT EXISTS client_request (
    id UUID PRIMARY KEY,
    method TEXT NOT NULL,
    host TEXT NOT NULL,
    path TEXT NOT NULL,
    request_body TEXT,
    request_response TEXT,
    status_code INT,
    error TEXT,
    duration_ms BIGINT,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_client_request_requested_at ON client_request (requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_client_request_host ON client_request (host);
