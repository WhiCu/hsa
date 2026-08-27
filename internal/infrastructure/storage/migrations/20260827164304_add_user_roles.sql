-- +goose Up
-- +goose StatementBegin

CREATE TYPE user_role AS ENUM ('member', 'admin');
ALTER TABLE users ADD COLUMN role user_role NOT NULL DEFAULT 'member';
UPDATE users SET role = 'admin' WHERE invited_by IS NULL;

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

ALTER TABLE users DROP COLUMN IF EXISTS role;
DROP TYPE IF EXISTS user_role;

-- +goose StatementEnd