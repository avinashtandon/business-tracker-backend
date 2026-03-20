CREATE TABLE trades (
    id BINARY(16) NOT NULL PRIMARY KEY,
    user_id BINARY(16) NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    position VARCHAR(50) NOT NULL,
    buying_price DECIMAL(15,2) NOT NULL,
    buying_date DATE NOT NULL,
    selling_price DECIMAL(15,2) NULL,
    selling_date DATE NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'Open',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_trades_user_id (user_id),
    CONSTRAINT fk_trades_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
