CREATE TABLE audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID REFERENCES users(id),
    event_type VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),
    entity_id UUID,
    action VARCHAR(50) NOT NULL,
    result VARCHAR(30) NOT NULL,
    ip_address INET,
    correlation_id UUID NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_events_actor_user_id ON audit_events (actor_user_id);
CREATE INDEX idx_audit_events_event_type_created_at ON audit_events (event_type, created_at);
CREATE INDEX idx_audit_events_correlation_id ON audit_events (correlation_id);