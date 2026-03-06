CREATE TABLE loans (
    id BINARY(16) NOT NULL PRIMARY KEY,
    user_id BINARY(16) NOT NULL,
    person_name VARCHAR(255) NOT NULL,
    purpose VARCHAR(255) NOT NULL,
    principal_amount DECIMAL(15,2) NOT NULL,
    interest_amount DECIMAL(15,2) NOT NULL,
    duration VARCHAR(100) NOT NULL,
    due_date DATE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_loans_user_id (user_id),
    CONSTRAINT fk_loans_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
