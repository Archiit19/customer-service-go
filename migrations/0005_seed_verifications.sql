BEGIN;

CREATE OR REPLACE FUNCTION generate_random_pan()
    RETURNS TEXT AS $$
DECLARE
letters  TEXT := 'ABCDEFGHIJKLMNOPQRSTUVWXYZ';
    pan TEXT := '';
    i INT;
BEGIN
    -- Generate first 5 letters
FOR i IN 1..5 LOOP
            pan := pan || substr(letters, floor(random() * 26 + 1)::int, 1);
END LOOP;

    -- Generate next 4 digits
FOR i IN 1..4 LOOP
            pan := pan || floor(random() * 10)::int;
END LOOP;

    -- Add last letter
    pan := pan || substr(letters, floor(random() * 26 + 1)::int, 1);

RETURN pan;
END;
$$ LANGUAGE plpgsql VOLATILE;

-- Unique generator: keeps looping until an unused PAN is found
CREATE OR REPLACE FUNCTION generate_unique_pan()
    RETURNS TEXT AS $$
DECLARE
new_pan TEXT;
BEGIN
    LOOP
new_pan := generate_random_pan();
        EXIT WHEN NOT EXISTS (SELECT 1 FROM verifications WHERE pan_number = new_pan);
END LOOP;
RETURN new_pan;
END;
$$ LANGUAGE plpgsql VOLATILE;

-- Link customers to verification statuses

INSERT INTO verifications (customer_id, pan_number)
SELECT id,
       generate_unique_pan()
FROM customers;


COMMIT;