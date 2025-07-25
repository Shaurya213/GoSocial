CREATE TABLE IF NOT EXISTS messages (
                                        id BIGINT AUTO_INCREMENT PRIMARY KEY,
                                        conversation_id BIGINT NOT NULL,
                                        sender_id BIGINT NOT NULL,
                                        body TEXT,
                                        sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    -- Add indices and FKs as appropriate
                                        FOREIGN KEY (conversation_id) REFERENCES conversations(id),
                                        FOREIGN KEY (sender_id) REFERENCES users(user_id)
);
