-- =====================================================
-- Add Optimistic Locking Version Columns
-- =====================================================
-- This migration adds version columns to publishers and
-- bidders tables to prevent lost updates from concurrent
-- modifications using optimistic locking.
-- =====================================================

-- Add version column to publishers table
ALTER TABLE publishers
ADD COLUMN version INTEGER NOT NULL DEFAULT 1;

-- Add version column to bidders table
ALTER TABLE bidders
ADD COLUMN version INTEGER NOT NULL DEFAULT 1;

-- Create function to automatically increment version on update
CREATE OR REPLACE FUNCTION increment_version()
RETURNS TRIGGER AS $$
BEGIN
    NEW.version = OLD.version + 1;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to increment version on publishers update
CREATE TRIGGER trigger_publishers_version
    BEFORE UPDATE ON publishers
    FOR EACH ROW
    EXECUTE FUNCTION increment_version();

-- Create trigger to increment version on bidders update
CREATE TRIGGER trigger_bidders_version
    BEFORE UPDATE ON bidders
    FOR EACH ROW
    EXECUTE FUNCTION increment_version();

COMMENT ON COLUMN publishers.version IS 'Optimistic locking version - increments on each update';
COMMENT ON COLUMN bidders.version IS 'Optimistic locking version - increments on each update';
