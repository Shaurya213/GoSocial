CREATE TABLE IF NOT EXISTS conversations (
                                             id BIGINT AUTO_INCREMENT PRIMARY KEY,
                                             type ENUM('private', 'group') NOT NULL,
                                             created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    -- Add group name or extra metadata if needed
    -- created_by BIGINT, -- (optional, for groups)
    -- INDEXes for searching/joining as needed
);
