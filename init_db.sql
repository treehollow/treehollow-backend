create table user_info
(
    email_hash CHAR(64) NOT NULL,
    token      CHAR(32) NOT NULL,
    timestamp  INT      NOT NULL,
    PRIMARY KEY (email_hash),
    INDEX (token)
) DEFAULT CHARSET = ascii;

create table verification_codes
(
    email_hash CHAR(64)    NOT NULL,
    timestamp  INT         NOT NULL,
    code       VARCHAR(20) NOT NULL,
    PRIMARY KEY (email_hash)
) DEFAULT CHARSET = ascii;

create table posts
(
    pid        INT            NOT NULL AUTO_INCREMENT,
    email_hash CHAR(64)       NOT NULL,
    text       VARCHAR(10000) NOT NULL,
    timestamp  INT            NOT NULL,
    tag        VARCHAR(60)    NOT NULL,
    type       VARCHAR(60)    NOT NULL,
    file_path  VARCHAR(60)    NOT NULL,
    likenum    INT            NOT NULL,
    replynum   INT            NOT NULL,
    PRIMARY KEY (pid)
) DEFAULT CHARSET = utf8mb4;

ALTER TABLE posts
    ADD COLUMN reportnum INT DEFAULT 0 AFTER replynum;

ALTER TABLE posts
    ADD FULLTEXT INDEX (text) WITH PARSER ngram;

ALTER TABLE posts
    ADD INDEX (tag);

create table comments
(
    cid        INT            NOT NULL AUTO_INCREMENT,
    pid        INT            NOT NULL,
    email_hash CHAR(64)       NOT NULL,
    text       VARCHAR(10000) NOT NULL,
    tag        VARCHAR(60)    NOT NULL,
    timestamp  INT            NOT NULL,
    name       VARCHAR(60)    NOT NULL,
    PRIMARY KEY (cid),
    INDEX (pid)
) DEFAULT CHARSET = utf8mb4;

create table reports
(
    rid        INT            NOT NULL AUTO_INCREMENT,
    pid        INT            NOT NULL,
    email_hash CHAR(64)       NOT NULL,
    reason     VARCHAR(10000) NOT NULL,
    timestamp  INT            NOT NULL,
    PRIMARY KEY (rid),
    CONSTRAINT pid_email UNIQUE (pid, email_hash),
    INDEX (pid)
) DEFAULT CHARSET = utf8mb4;

create table banned
(
    email_hash  CHAR(64)       NOT NULL,
    reason      VARCHAR(10000) NOT NULL,
    timestamp   INT            NOT NULL,
    expire_time INT            NOT NULL,
    INDEX (email_hash)
) DEFAULT CHARSET = utf8mb4;

ALTER TABLE banned
    MODIFY COLUMN reason VARCHAR(11000) NOT NULL;

create table attentions
(
    email_hash CHAR(64) NOT NULL,
    pid        INT      NOT NULL,
    INDEX (email_hash),
    CONSTRAINT pid_email UNIQUE (pid, email_hash)
) DEFAULT CHARSET = ascii;

ALTER TABLE banned
    ADD INDEX (timestamp);

ALTER TABLE reports
    ADD INDEX (timestamp);

ALTER TABLE verification_codes
    ADD COLUMN failed_times INT DEFAULT 0 AFTER code;