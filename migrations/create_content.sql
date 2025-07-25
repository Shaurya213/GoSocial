CREATE TABLE IF NOT EXISTS contents (
                                        content_id BIGINT AUTO_INCREMENT PRIMARY KEY,
                                        author_id BIGINT NOT NULL,
                                        type ENUM('POST', 'STORY', 'REEL') NOT NULL,
    text_content TEXT,
    media_ref_id BIGINT,
    privacy ENUM('public', 'friends', 'private') NOT NULL,
    expiration DATETIME,
    duration INT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    FOREIGN KEY (author_id) REFERENCES users(user_id),
    FOREIGN KEY (media_ref_id) REFERENCES media_refs(media_ref_id)
    );
