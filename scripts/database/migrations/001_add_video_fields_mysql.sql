-- MySQL Migration: Add video-specific fields to stored requests
-- Migration: 001_add_video_fields
-- Database: MySQL
-- Date: 2026-01-22

-- Add video-specific fields to stored_requests table
ALTER TABLE stored_requests
  ADD COLUMN IF NOT EXISTS video_duration_min INT,
  ADD COLUMN IF NOT EXISTS video_duration_max INT,
  ADD COLUMN IF NOT EXISTS video_protocols JSON,
  ADD COLUMN IF NOT EXISTS video_start_delay INT,
  ADD COLUMN IF NOT EXISTS video_mimes JSON,
  ADD COLUMN IF NOT EXISTS video_skippable BOOLEAN DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS video_skip_delay INT;

-- Add video-specific fields to stored_imps table
ALTER TABLE stored_imps
  ADD COLUMN IF NOT EXISTS video_duration_min INT,
  ADD COLUMN IF NOT EXISTS video_duration_max INT,
  ADD COLUMN IF NOT EXISTS video_protocols JSON,
  ADD COLUMN IF NOT EXISTS video_start_delay INT,
  ADD COLUMN IF NOT EXISTS video_mimes JSON,
  ADD COLUMN IF NOT EXISTS video_skippable BOOLEAN DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS video_skip_delay INT;

-- Add comments for documentation (MySQL 8.0+)
ALTER TABLE stored_requests
  MODIFY COLUMN video_duration_min INT COMMENT 'Minimum video duration in seconds',
  MODIFY COLUMN video_duration_max INT COMMENT 'Maximum video duration in seconds',
  MODIFY COLUMN video_protocols JSON COMMENT 'Array of supported video protocols (OpenRTB enum)',
  MODIFY COLUMN video_start_delay INT COMMENT 'Start delay in seconds (-1=pre-roll, 0=mid-roll, >0=post-roll)',
  MODIFY COLUMN video_mimes JSON COMMENT 'Array of supported MIME types (e.g., video/mp4, video/webm)',
  MODIFY COLUMN video_skippable BOOLEAN COMMENT 'Whether the video ad is skippable',
  MODIFY COLUMN video_skip_delay INT COMMENT 'Delay before skip button appears (seconds)';

ALTER TABLE stored_imps
  MODIFY COLUMN video_duration_min INT COMMENT 'Minimum video duration in seconds',
  MODIFY COLUMN video_duration_max INT COMMENT 'Maximum video duration in seconds',
  MODIFY COLUMN video_protocols JSON COMMENT 'Array of supported video protocols (OpenRTB enum)',
  MODIFY COLUMN video_start_delay INT COMMENT 'Start delay in seconds (-1=pre-roll, 0=mid-roll, >0=post-roll)',
  MODIFY COLUMN video_mimes JSON COMMENT 'Array of supported MIME types (e.g., video/mp4, video/webm)',
  MODIFY COLUMN video_skippable BOOLEAN COMMENT 'Whether the video ad is skippable',
  MODIFY COLUMN video_skip_delay INT COMMENT 'Delay before skip button appears (seconds)';

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_stored_requests_video_duration ON stored_requests(video_duration_min, video_duration_max);
CREATE INDEX IF NOT EXISTS idx_stored_imps_video_duration ON stored_imps(video_duration_min, video_duration_max);
