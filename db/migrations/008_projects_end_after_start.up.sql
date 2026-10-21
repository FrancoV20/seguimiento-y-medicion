DO $$
DECLARE c text;
BEGIN
    SELECT conname INTO c FROM pg_constraint
    WHERE conrelid = 'projects'::regclass AND contype = 'c'
      AND pg_get_constraintdef(oid) LIKE '%end_date >= start_date%';
    IF c IS NOT NULL THEN
        EXECUTE format('ALTER TABLE projects DROP CONSTRAINT %I', c);
    END IF;
END $$;

ALTER TABLE projects
    ADD CONSTRAINT projects_end_after_start CHECK (end_date > start_date);
