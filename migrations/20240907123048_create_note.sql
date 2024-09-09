-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS notes
(
    id           BIGSERIAL   NOT NULL PRIMARY KEY,
    user_id      BIGINT      NOT NULL,
    name         VARCHAR(50) NOT NULL,
    description  VARCHAR(255),
    is_completed BOOLEAN     NOT NULL DEFAULT 'FALSE',
    created_at   DATE        NOT NULL DEFAULT NOW()::DATE,
    deadline_at  DATE,
    CONSTRAINT notes_to_users_id_fk FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS notes CASCADE;
-- +goose StatementEnd
