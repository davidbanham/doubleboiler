CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID not null,
    organisation_id UUID not null,
    table_name text not null,
    stamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (now() at time zone 'utc'),
    user_id text DEFAULT current_setting('application_name'),
    action TEXT NOT NULL CHECK (action IN ('D','U')),
    old_row_data jsonb
);
