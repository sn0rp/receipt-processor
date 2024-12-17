ALTER TABLE items
    ADD CONSTRAINT fk_receipt
    FOREIGN KEY (receipt_id)
    REFERENCES receipts(id)
    ON DELETE CASCADE; 