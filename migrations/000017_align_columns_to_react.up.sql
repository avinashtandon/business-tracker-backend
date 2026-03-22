ALTER TABLE trades
    CHANGE buy_price buying_price BIGINT NOT NULL,
    CHANGE buy_date buying_date TIMESTAMP NOT NULL,
    CHANGE sell_price selling_price BIGINT NULL,
    CHANGE sell_date selling_date TIMESTAMP NULL;
