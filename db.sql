DROP TABLE IF EXISTS `chat`;
CREATE TABLE IF NOT EXISTS `chat`
(
    `id`    VARCHAR(255) PRIMARY KEY NOT NULL,
    `active`  SMALLINT     NOT NULL DEFAULT 0,
    `exclude` TEXT,
    `url`     TEXT,
    `created` DATETIME     NOT NULL,
    `updated` DATETIME     NOT NULL
);

/*
id - unique chat identifier
active - flat chat is active or not
exclude - list of excluded members
url - chat URL for calls
created - timestamp of item create
updated - timestamp of item update
 */