-- PostgreSQL Migration: Add video-specific fields to stored requests
-- Migration: 001_add_video_fields
-- Database: PostgreSQL
-- Date: 2026-01-22

-- Add video-specific fields to stored_requests table
ALTER TABLE stored_requests ADD COLUMN IF NOT EXISTS video_duration_min INTEGER;
ALTER TABLE stored_requests ADD COLUMN IF NOT EXISTS video_duration_max INTEGER;
ALTER TABLE stored_requests ADD COLUMN IF NOT EXISTS video_protocols INTEGER[];
ALTER TABLE stored_requests ADD COLUMN IF NOT EXISTS video_start_delay INTEGER;
ALTER TABLE stored_requests ADD COLUMN IF NOT EXISTS video_mimes TEXT[];
ALTER TABLE stored_requests ADD COLUMN IF NOT EXISTS video_skippable BOOLEAN DEFAULT FALSE;
ALTER TABLE stored_requests ADD COLUMN IF NOT EXISTS video_skip_delay INTEGER;

-- Add video-specific fields to stored_imps table
ALTER TABLE stored_imps ADD COLUMN IF NOT EXISTS video_duration_min INTEGER;
ALTER TABLE stored_imps ADD COLUMN IF NOT EXISTS video_duration_max INTEGER;
ALTER TABLE stored_imps ADD COLUMN IF NOT EXISTS video_protocols INTEGER[];
ALTER TABLE stored_imps ADD COLUMN IF NOT EXISTS video_start_delay INTEGER;
ALTER TABLE stored_imps ADD COLUMN IF NOT EXISTS video_mimes TEXT[];
ALTER TABLE stored_imps ADD COLUMN IF NOT EXISTS video_skippable BOOLEAN DEFAULT FALSE;
ALTER TABLE stored_imps ADD COLUMN IF NOT EXISTS video_skip_delay INTEGER;

-- Add comments for documentation
COMMENT ON COLUMN stored_requests.video_duration_min IS 'Minimum video duration in seconds';
COMMENT ON COLUMN stored_requests.video_duration_max IS 'Maximum video duration in seconds';
COMMENT ON COLUMN stored_requests.video_protocols IS 'Array of supported video protocols (OpenRTB enum)';
COMMENT ON COLUMN stored_requests.video_start_delay IS 'Start delay in seconds (-1=pre-roll, 0=mid-roll, >0=post-roll)';
COMMENT ON COLUMN stored_requests.video_mimes IS 'Array of supported MIME types (e.g., video/mp4, video/webm)';
COMMENT ON COLUMN stored_requests.video_skippable IS 'Whether the video ad is skippable';
COMMENT ON COLUMN stored_requests.video_skip_delay IS 'Delay before skip button appears (seconds)';

COMMENT ON COLUMN stored_imps.video_duration_min IS 'Minimum video duration in seconds';
COMMENT ON COLUMN stored_imps.video_duration_max IS 'Maximum video duration in seconds';
COMMENT ON COLUMN stored_imps.video_protocols IS 'Array of supported video protocols (OpenRTB enum)';
COMMENT ON COLUMN stored_imps.video_start_delay IS 'Start delay in seconds (-1=pre-roll, 0=mid-roll, >0=post-roll)';
COMMENT ON COLUMN stored_imps.video_mimes IS 'Array of supported MIME types (e.g., video/mp4, video/webm)';
COMMENT ON COLUMN stored_imps.video_skippable IS 'Whether the video ad is skippable';
COMMENT ON COLUMN stored_imps.video_skip_delay IS 'Delay before skip button appears (seconds)';

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_stored_requests_video_duration ON stored_requests(video_duration_min, video_duration_max);
CREATE INDEX IF NOT EXISTS idx_stored_requests_video_protocols ON stored_requests USING GIN(video_protocols);
CREATE INDEX IF NOT EXISTS idx_stored_requests_video_mimes ON stored_requests USING GIN(video_mimes);

CREATE INDEX IF NOT EXISTS idx_stored_imps_video_duration ON stored_imps(video_duration_min, video_duration_max);
CREATE INDEX IF NOT EXISTS idx_stored_imps_video_protocols ON stored_imps USING GIN(video_protocols);
CREATE INDEX IF NOT EXISTS idx_stored_imps_video_mimes ON stored_imps USING GIN(video_mimes);
