create table user_info
(
    email_hash CHAR(64) NOT NULL,
    token      CHAR(32) NOT NULL,
    timestamp  INT      NOT NULL,
    attentions VARCHAR(10000),
    PRIMARY KEY (email_hash),
    INDEX (token)
) DEFAULT CHARSET = ascii;

create table verification_codes
(
    email_hash CHAR(64) NOT NULL,
    timestamp  INT      NOT NULL,
    code       INT      NOT NULL,
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
