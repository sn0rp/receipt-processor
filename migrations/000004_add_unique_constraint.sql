CREATE UNIQUE INDEX idx_unique_receipt ON receipts (
    retailer,
    purchase_date,
    purchase_time,
    total,
    MD5(CAST((
        SELECT string_agg(short_description || ':' || price, ',' ORDER BY short_description)
        FROM items
        WHERE items.receipt_id = receipts.id
    ) AS text))
);
