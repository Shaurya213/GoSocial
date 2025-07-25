CREATE TABLE IF NOT EXISTS reactions (
                                         reaction_id BIGINT AUTO_INCREMENT PRIMARY KEY,
                                         user_id BIGINT NOT NULL,
                                         content_id BIGINT NOT NULL,
                                         reaction_type ENUM('like', 'love') NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(user_id),
    FOREIGN KEY (content_id) REFERENCES contents(content_id)
    );
