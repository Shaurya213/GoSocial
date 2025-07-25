CREATE TABLE IF NOT EXISTS notifications (
                                             notification_id BIGINT AUTO_INCREMENT PRIMARY KEY,
                                             user_id BIGINT NOT NULL,
                                             content TEXT NOT NULL,
                                             type ENUM('system', 'friend', 'reaction', 'chat') NOT NULL,
    status ENUM('sent', 'delivered', 'read') DEFAULT 'sent',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    read_at DATETIME,

    FOREIGN KEY (user_id) REFERENCES users(user_id)
    );
