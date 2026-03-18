CREATE TABLE templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    channel channel_type NOT NULL,
    subject VARCHAR(500),
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE notifications
    ADD CONSTRAINT fk_notifications_template_id
    FOREIGN KEY (template_id) REFERENCES templates(id);
