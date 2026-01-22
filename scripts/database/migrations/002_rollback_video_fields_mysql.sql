-- MySQL Rollback: Remove video-specific fields from stored requests
-- Migration Rollback: 001_add_video_fields
-- Database: MySQL
-- Date: 2026-01-22

-- Drop indexes first
DROP INDEX IF EXISTS idx_stored_imps_video_duration ON stored_imps;
DROP INDEX IF EXISTS idx_stored_requests_video_duration ON stored_requests;

-- Remove video-specific fields from stored_imps table
ALTER TABLE stored_imps
  DROP COLUMN IF EXISTS video_skip_delay,
  DROP COLUMN IF EXISTS video_skippable,
  DROP COLUMN IF EXISTS video_mimes,
  DROP COLUMN IF EXISTS video_start_delay,
  DROP COLUMN IF EXISTS video_protocols,
  DROP COLUMN IF EXISTS video_duration_max,
  DROP COLUMN IF EXISTS video_duration_min;

-- Remove video-specific fields from stored_requests table
ALTER TABLE stored_requests
  DROP COLUMN IF EXISTS video_skip_delay,
  DROP COLUMN IF EXISTS video_skippable,
  DROP COLUMN IF EXISTS video_mimes,
  DROP COLUMN IF EXISTS video_start_delay,
  DROP COLUMN IF EXISTS video_protocols,
  DROP COLUMN IF EXISTS video_duration_max,
  DROP COLUMN IF EXISTS video_duration_min;
