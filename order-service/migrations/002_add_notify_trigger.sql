CREATE OR REPLACE FUNCTION notify_order_status_change()
RETURNS TRIGGER AS $$
BEGIN
  PERFORM pg_notify(
    'order_status_channel',
    json_build_object('order_id', NEW.id, 'status', NEW.status)::text
  );
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

ALTER TABLE orders ADD COLUMN status TEXT DEFAULT 'Pending';

CREATE TRIGGER order_status_change_trigger
    AFTER UPDATE OF status ON orders
    FOR EACH ROW
    EXECUTE FUNCTION notify_order_status_change();