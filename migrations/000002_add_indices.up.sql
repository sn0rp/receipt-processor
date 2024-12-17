CREATE INDEX idx_receipts_created_at ON receipts(created_at);
CREATE INDEX idx_items_receipt_id ON items(receipt_id); 