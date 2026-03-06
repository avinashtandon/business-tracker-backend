CREATE TABLE crypto_holdings (
  id         BINARY(16) PRIMARY KEY,
  user_id    BINARY(16) NOT NULL,
  name       VARCHAR(100) NOT NULL,
  symbol     VARCHAR(20),
  created_at DATETIME,
  INDEX idx_crypto_holdings_user_id (user_id),
  CONSTRAINT fk_crypto_holdings_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE crypto_purchases (
  id              BINARY(16) PRIMARY KEY,
  holding_id      BINARY(16) NOT NULL,
  quantity        DECIMAL(20,8) NOT NULL,
  buy_price       DECIMAL(20,2) NOT NULL,
  invested_amount DECIMAL(20,2) NOT NULL,
  date            DATE NOT NULL,
  exchange        VARCHAR(100),
  note            VARCHAR(255),
  created_at      DATETIME,
  INDEX idx_crypto_purchases_holding_id (holding_id),
  CONSTRAINT fk_crypto_purchases_holding_id FOREIGN KEY (holding_id) REFERENCES crypto_holdings (id) ON DELETE CASCADE
);
