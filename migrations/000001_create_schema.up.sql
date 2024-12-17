CREATE TABLE receipts (
    id VARCHAR(36) PRIMARY KEY,
    retailer VARCHAR(255) NOT NULL,
    purchase_date DATE NOT NULL,
    purchase_time TIME NOT NULL,
    total DECIMAL(10,2) NOT NULL,
    points INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    receipt_id VARCHAR(36) NOT NULL,
    short_description VARCHAR(255) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
); 