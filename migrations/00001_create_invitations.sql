-- +goose Up
-- +goose StatementBegin
CREATE TABLE invitations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    guest_name TEXT NOT NULL,
    phone TEXT NOT NULL UNIQUE,
    token TEXT NOT NULL UNIQUE,
    invite_message TEXT,
    sent_at DATETIME NULL,
    opened_at DATETIME NULL,
    responded_at DATETIME NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_invitations_token ON invitations(token);
CREATE INDEX idx_invitations_phone ON invitations(phone);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_invitations_phone;
DROP INDEX IF EXISTS idx_invitations_token;
DROP TABLE IF EXISTS invitations;
-- +goose StatementEnd

