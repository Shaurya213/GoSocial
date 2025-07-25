CREATE TABLE IF NOT EXISTS friends (
                                       id BIGINT AUTO_INCREMENT PRIMARY KEY,
                                       user_id BIGINT NOT NULL,
                                       friend_user_id BIGINT NOT NULL,
                                       status ENUM('pending', 'accepted', 'blocked') NOT NULL,
    requested_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    accepted_at DATETIME NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(user_id),
    FOREIGN KEY (friend_user_id) REFERENCES users(user_id)
    );
