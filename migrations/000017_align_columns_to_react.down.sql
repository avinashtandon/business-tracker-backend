ALTER TABLE trades
    CHANGE buying_price buy_price BIGINT NOT NULL,
    CHANGE buying_date buy_date TIMESTAMP NOT NULL,
    CHANGE selling_price sell_price BIGINT NULL,
    CHANGE selling_date sell_date TIMESTAMP NULL;
