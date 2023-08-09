DROP TABLE IF EXISTS `chat`;
CREATE TABLE IF NOT EXISTS `chat`
(
    `id`       VARCHAR(255) PRIMARY KEY NOT NULL,
    `active`   SMALLINT                 NOT NULL DEFAULT 0,
    `gpt`      SMALLINT                 NOT NULL DEFAULT 0,
    `exclude`  TEXT,
    `url`      TEXT,
    `url_text` VARCHAR(255)             NOT NULL DEFAULT 'call',
    `created`  DATETIME                 NOT NULL,
    `updated`  DATETIME                 NOT NULL
);

/*
id - unique chat identifier
active - chat is active or not
gpt - allow ChatGPT requests
exclude - list of excluded users
url - chat URL for calls
url_text - text for chat URL
created - timestamp of item create
updated - timestamp of item update

Migrations:
ALTER TABLE `chat` ADD COLUMN `url_text` VARCHAR(255) NOT NULL DEFAULT 'call';
ALTER TABLE `chat` ADD COLUMN `gpt` SMALLINT NOT NULL DEFAULT 0;
 */

