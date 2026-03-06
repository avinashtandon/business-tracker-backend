CREATE TABLE transactions (
    id BINARY(16) NOT NULL PRIMARY KEY,
    loan_id BINARY(16) NOT NULL,
    date DATE NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    mode VARCHAR(50) NOT NULL,
    note TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_transactions_loan_id (loan_id),
    CONSTRAINT fk_transactions_loan_id FOREIGN KEY (loan_id) REFERENCES loans (id) ON DELETE CASCADE
);
