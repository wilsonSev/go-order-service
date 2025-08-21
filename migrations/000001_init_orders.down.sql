DROP TRIGGER IF EXISTS trg_orders_set_updated_at ON orders;
DROP FUNCTION IF EXISTS set_updated_at();
DROP TABLE IF EXISTS orders;