CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE channel_type AS ENUM ('sms', 'email', 'push');
CREATE TYPE priority_type AS ENUM ('high', 'normal', 'low');
CREATE TYPE status_type AS ENUM ('pending', 'scheduled', 'queued', 'processing', 'delivered', 'failed', 'cancelled');

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key VARCHAR(255),
    batch_id UUID,
    channel channel_type NOT NULL,
    priority priority_type NOT NULL DEFAULT 'normal',
    recipient VARCHAR(500) NOT NULL,
    subject VARCHAR(500),
    body TEXT NOT NULL,
    template_id UUID,
    template_vars JSONB,
    status status_type NOT NULL DEFAULT 'pending',
    scheduled_at TIMESTAMPTZ,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 5,
    last_error TEXT,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_notifications_idempotency_key
    ON notifications (idempotency_key) WHERE idempotency_key IS NOT NULL;

CREATE INDEX idx_notifications_status ON notifications (status);
CREATE INDEX idx_notifications_batch_id ON notifications (batch_id) WHERE batch_id IS NOT NULL;
CREATE INDEX idx_notifications_scheduled_at
    ON notifications (scheduled_at) WHERE status = 'scheduled' AND scheduled_at IS NOT NULL;
CREATE INDEX idx_notifications_created_at ON notifications (created_at);
CREATE INDEX idx_notifications_channel_status ON notifications (channel, status);
