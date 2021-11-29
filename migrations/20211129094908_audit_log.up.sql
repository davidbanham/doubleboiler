CREATE extension IF NOT EXISTS "uuid-ossp";
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_id UUID not null,
    table_name text not null,
    action_tstamp_tx TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (now() at time zone 'utc'),
    user_id text DEFAULT current_setting('application_name'),
    action TEXT NOT NULL CHECK (action IN ('D','U')),
    old_row_data jsonb
);
