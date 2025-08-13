-- +goose Up
-- SQL to create users table

CREATE TABLE users (
                       user_id BIGINT AUTO_INCREMENT PRIMARY KEY,
                       handle VARCHAR(50) NOT NULL UNIQUE,
                       password_hash VARCHAR(255) NOT NULL,
                       profile_details TEXT,
                       email VARCHAR(255),
                       phone VARCHAR(20),
                       status ENUM('active', 'banned', 'deleted') DEFAULT 'active',
                       created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                       updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                       deleted_at DATETIME NULL,

                       INDEX idx_handle (handle),
                       INDEX idx_status (status)
);

-- +goose Down
-- SQL to drop users table

DROP TABLE IF EXISTS users;
