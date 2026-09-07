-- ITCodeX 系统表（gf gen dao 源表）。标识符使用 MySQL 反引号。
-- 表前缀 c_ 与 metadata.tablePrefix 默认值一致。

CREATE TABLE IF NOT EXISTS `c_collections` (
    `id` BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `name` VARCHAR(255) NOT NULL UNIQUE,
    `display_name` VARCHAR(255) NOT NULL DEFAULT '',
    `type` VARCHAR(50) NOT NULL DEFAULT 'general',
    `options` JSON NULL,
    `created_at` DATETIME NULL,
    `updated_at` DATETIME NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `c_fields` (
    `id` BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `collection_name` VARCHAR(255) NOT NULL,
    `name` VARCHAR(255) NOT NULL,
    `type` VARCHAR(50) NOT NULL,
    `display_name` VARCHAR(255) NOT NULL DEFAULT '',
    `is_required` TINYINT(1) NOT NULL DEFAULT 0,
    `is_unique` TINYINT(1) NOT NULL DEFAULT 0,
    `is_indexed` TINYINT(1) NOT NULL DEFAULT 0,
    `validation` JSON NULL,
    `options` JSON NULL,
    `sort` INT NOT NULL DEFAULT 0,
    `created_at` DATETIME NULL,
    UNIQUE KEY `uk_collection_field` (`collection_name`, `name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `c_indexes` (
    `id` BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `collection_name` VARCHAR(255) NOT NULL,
    `name` VARCHAR(255) NOT NULL,
    `fields` JSON NOT NULL,
    `unique` TINYINT(1) NOT NULL DEFAULT 0,
    `options` JSON NULL,
    `created_at` DATETIME NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `c_yaegi_scripts` (
    `id` BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `collection_name` VARCHAR(255) NULL,
    `name` VARCHAR(255) NOT NULL,
    `hook_point` VARCHAR(50) NOT NULL,
    `content` LONGTEXT NOT NULL,
    `api_path` VARCHAR(255) NULL,
    `http_method` VARCHAR(20) NULL,
    `enabled` TINYINT(1) NOT NULL DEFAULT 1,
    `priority` INT NOT NULL DEFAULT 0,
    `options` JSON NULL,
    `created_at` DATETIME NULL,
    `updated_at` DATETIME NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
