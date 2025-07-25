CREATE TABLE media_refs (
                            media_ref_id BIGINT AUTO_INCREMENT PRIMARY KEY,
                            type ENUM('image', 'video') NOT NULL,
                            file_path VARCHAR(255) NOT NULL,
                            file_name VARCHAR(255) NOT NULL,
                            uploaded_by BIGINT NOT NULL,
                            uploaded_at DATETIME NOT NULL,
                            size_bytes BIGINT NOT NULL,
                            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                            deleted_at TIMESTAMP NULL,

                            INDEX idx_uploaded_by (uploaded_by),
                            INDEX idx_type (type),
                            INDEX idx_file_path (file_path)
);
