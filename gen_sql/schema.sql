CREATE TABLE IF NOT EXISTS users
(
    id       BIGSERIAL   NOT NULL PRIMARY KEY,
    login    VARCHAR(20) NOT NULL,
    password VARCHAR(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS notes
(
    id           BIGSERIAL   NOT NULL PRIMARY KEY,
    user_id      BIGINT      NOT NULL,
    name         VARCHAR(50) NOT NULL,
    description  VARCHAR(255),
    is_completed BOOLEAN     NOT NULL DEFAULT 'FALSE',
    created_at   TIMESTAMP   NOT NULL DEFAULT NOW(),
    deadline_at  TIMESTAMP,
    CONSTRAINT notes_to_users_id_fk FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
);