ALTER TABLE trades
    CHANGE buy_price buying_price DOUBLE NOT NULL,
    CHANGE buy_date buying_date VARCHAR(255) NOT NULL,
    CHANGE sell_price selling_price DOUBLE NULL,
    CHANGE sell_date selling_date VARCHAR(255) NULL;
