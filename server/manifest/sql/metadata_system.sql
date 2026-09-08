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

CREATE TABLE IF NOT EXISTS `c_auth_users` (
    `id` BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `username` VARCHAR(191) NOT NULL UNIQUE,
    `display_name` VARCHAR(255) NOT NULL DEFAULT '',
    `password_hash` VARCHAR(512) NOT NULL,
    `enabled` TINYINT(1) NOT NULL DEFAULT 1,
    `created_at` DATETIME NULL,
    `updated_at` DATETIME NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `c_auth_roles` (
    `id` BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `name` VARCHAR(191) NOT NULL UNIQUE,
    `display_name` VARCHAR(255) NOT NULL DEFAULT '',
    `created_at` DATETIME NULL,
    `updated_at` DATETIME NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `c_auth_user_roles` (
    `user_id` BIGINT NOT NULL,
    `role_id` BIGINT NOT NULL,
    `created_at` DATETIME NULL,
    PRIMARY KEY (`user_id`, `role_id`),
    KEY `idx_auth_user_roles_role` (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `c_auth_sessions` (
    `id` VARCHAR(64) NOT NULL PRIMARY KEY,
    `user_id` BIGINT NOT NULL,
    `token_hash` CHAR(64) NOT NULL,
    `expires_at` DATETIME NOT NULL,
    `revoked_at` DATETIME NULL,
    `created_at` DATETIME NULL,
    KEY `idx_auth_sessions_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `c_acl_policies` (
    `id` BIGINT NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `subject_type` VARCHAR(20) NOT NULL,
    `subject` VARCHAR(191) NOT NULL DEFAULT '',
    `resource` VARCHAR(191) NOT NULL,
    `action` VARCHAR(100) NOT NULL,
    `row_filter` JSON NULL,
    `read_fields` JSON NULL,
    `write_fields` JSON NULL,
    `enabled` TINYINT(1) NOT NULL DEFAULT 1,
    `created_at` DATETIME NULL,
    `updated_at` DATETIME NULL,
    KEY `idx_acl_resource_action` (`resource`, `action`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
