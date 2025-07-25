CREATE TABLE IF NOT EXISTS devices (
                                       device_token VARCHAR(255) PRIMARY KEY,
    user_id BIGINT NOT NULL,
    platform VARCHAR(10) NOT NULL,
    registered_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_active DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(user_id)
    );
