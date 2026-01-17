-- +goose Up
-- +goose StatementBegin
ALTER TABLE responses ADD COLUMN menu_preference TEXT CHECK(menu_preference IN ('standard', 'vegan', ''));
ALTER TABLE responses ADD COLUMN companion_menu_preference TEXT CHECK(companion_menu_preference IN ('standard', 'vegan', ''));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE responses DROP COLUMN companion_menu_preference;
ALTER TABLE responses DROP COLUMN menu_preference;
-- +goose StatementEnd

