-- PostgreSQL Rollback: Remove video-specific fields from stored requests
-- Migration Rollback: 001_add_video_fields
-- Database: PostgreSQL
-- Date: 2026-01-22

-- Drop indexes first
DROP INDEX IF EXISTS idx_stored_imps_video_mimes;
DROP INDEX IF EXISTS idx_stored_imps_video_protocols;
DROP INDEX IF EXISTS idx_stored_imps_video_duration;

DROP INDEX IF EXISTS idx_stored_requests_video_mimes;
DROP INDEX IF EXISTS idx_stored_requests_video_protocols;
DROP INDEX IF EXISTS idx_stored_requests_video_duration;

-- Remove video-specific fields from stored_imps table
ALTER TABLE stored_imps DROP COLUMN IF EXISTS video_skip_delay;
ALTER TABLE stored_imps DROP COLUMN IF EXISTS video_skippable;
ALTER TABLE stored_imps DROP COLUMN IF EXISTS video_mimes;
ALTER TABLE stored_imps DROP COLUMN IF EXISTS video_start_delay;
ALTER TABLE stored_imps DROP COLUMN IF EXISTS video_protocols;
ALTER TABLE stored_imps DROP COLUMN IF EXISTS video_duration_max;
ALTER TABLE stored_imps DROP COLUMN IF EXISTS video_duration_min;

-- Remove video-specific fields from stored_requests table
ALTER TABLE stored_requests DROP COLUMN IF EXISTS video_skip_delay;
ALTER TABLE stored_requests DROP COLUMN IF EXISTS video_skippable;
ALTER TABLE stored_requests DROP COLUMN IF EXISTS video_mimes;
ALTER TABLE stored_requests DROP COLUMN IF EXISTS video_start_delay;
ALTER TABLE stored_requests DROP COLUMN IF EXISTS video_protocols;
ALTER TABLE stored_requests DROP COLUMN IF EXISTS video_duration_max;
ALTER TABLE stored_requests DROP COLUMN IF EXISTS video_duration_min;
