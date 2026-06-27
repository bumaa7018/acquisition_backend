ALTER TABLE land_acquisition_assignee
    ADD COLUMN IF NOT EXISTS user_position TEXT NOT NULL DEFAULT '';
