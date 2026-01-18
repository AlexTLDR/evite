-- +goose Up
-- +goose StatementBegin
CREATE TABLE responses (
    id SERIAL PRIMARY KEY,
    invitation_id INTEGER NOT NULL,
    attending BOOLEAN NOT NULL,
    plus_one BOOLEAN NOT NULL,
    plus_one_name TEXT,
    guest_name_tag TEXT NOT NULL,
    kids_count INTEGER NOT NULL DEFAULT 0,
    comment TEXT,
    submitted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_latest BOOLEAN DEFAULT TRUE,
    FOREIGN KEY(invitation_id) REFERENCES invitations(id)
);

CREATE INDEX idx_responses_invitation_id ON responses(invitation_id);
CREATE INDEX idx_responses_is_latest ON responses(is_latest);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_responses_is_latest;
DROP INDEX IF EXISTS idx_responses_invitation_id;
DROP TABLE IF EXISTS responses;
-- +goose StatementEnd

