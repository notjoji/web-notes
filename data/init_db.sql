CREATE TABLE IF NOT EXISTS users
(
    id       BIGSERIAL   NOT NULL PRIMARY KEY,
    login    VARCHAR(20) NOT NULL,
    password VARCHAR(128) NOT NULL
);

INSERT INTO users (login, password)
VALUES ('123', 'a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3');

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

INSERT INTO notes (user_id, name, description, is_completed, deadline_at)
VALUES (1, 'Выбрать тему проекта', 'Наверное, заметки - это самое легкое', true, null),
       (1, 'Создать структуру БД', 'Пользователи и заметки', true, '2024-09-10'::DATE),
       (1, 'Создать все HTML-шаблоны', 'Страницы авторизации, регистрации, главная, создания и редактирования заметки',
        true, '2024-09-12'::DATE),
       (1, 'Написать API', 'Методы получения/создания/редактирования и удаления заметок + пользовательские методы',
        true, '2024-09-12'::DATE),
       (1, 'Развернуть проект в Docker', 'Подготовить Dockerfile и docker-compose файлы', true, '2024-09-13'::DATE),
       (1, 'Заняться дизайном страниц', 'Только не Bootstrap', false, null),
       (1, 'Написать Unit-тесты', 'На сервисную логику', false, '2024-09-14'::DATE),
       (1, 'Написать интеграционные тесты', 'Сначала разобраться, как их писать)', false, '2024-09-14'::DATE),
       (1, 'Проверить проект линтерами', 'Запустить golangci-lint', false, '2024-09-15'::DATE),
       (1, 'Запушить проект на Github', 'Разобраться с Github Actions', false, '2024-09-15'::DATE);
