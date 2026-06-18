-- Create "devices" table
CREATE TABLE `devices` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL,
  `phone` varchar(20) NULL,
  `session_data` longblob NULL,
  `telegram_user_id` bigint NULL,
  `telegram_first_name` varchar(100) NULL,
  `telegram_last_name` varchar(100) NULL,
  `telegram_phone` varchar(20) NULL,
  `avatar_color` varchar(20) NULL,
  `status` varchar(20) NULL DEFAULT 'no_session',
  `api_key` varchar(255) NULL,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_devices_api_key` (`api_key`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;
-- Create "logs" table
CREATE TABLE `logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `device_id` bigint unsigned NULL,
  `level` varchar(20) NULL DEFAULT 'info',
  `action` varchar(50) NULL,
  `message` text NULL,
  `created_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_logs_device_id` (`device_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;
-- Create "users" table
CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(50) NOT NULL,
  `password` varchar(255) NOT NULL,
  `name` varchar(100) NULL,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_users_username` (`username`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;
