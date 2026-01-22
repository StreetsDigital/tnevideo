# Database Migrations

This directory contains SQL migration scripts for Prebid Server database schema updates.

## Overview

These migrations add video-specific fields to the Prebid Server database schema to support CTV (Connected TV) and video advertising features.

## Migrations

### 001: Add Video Fields

**Files:**
- `001_add_video_fields_postgres.sql` - PostgreSQL migration
- `001_add_video_fields_mysql.sql` - MySQL migration

**Changes:**
- Adds video duration constraints (min/max)
- Adds video protocol support
- Adds video start delay configuration
- Adds MIME type specifications
- Adds skip functionality settings

**Tables Modified:**
- `stored_requests`
- `stored_imps`

**New Fields:**

| Field Name | Type | Description |
|------------|------|-------------|
| `video_duration_min` | INTEGER | Minimum acceptable video duration in seconds |
| `video_duration_max` | INTEGER | Maximum acceptable video duration in seconds |
| `video_protocols` | INTEGER[] / JSON | Supported video protocols (OpenRTB enum values) |
| `video_start_delay` | INTEGER | Ad start delay: -1=pre-roll, 0=mid-roll, >0=post-roll |
| `video_mimes` | TEXT[] / JSON | Supported MIME types (e.g., `video/mp4`, `video/webm`) |
| `video_skippable` | BOOLEAN | Whether the video ad can be skipped |
| `video_skip_delay` | INTEGER | Seconds before skip button appears |

### 002: Rollback Video Fields

**Files:**
- `002_rollback_video_fields_postgres.sql` - PostgreSQL rollback
- `002_rollback_video_fields_mysql.sql` - MySQL rollback

**Purpose:**
Rollback migration to remove video fields if needed.

## Running Migrations

### PostgreSQL

```bash
# Apply migration
psql -U <username> -d <database> -f scripts/database/migrations/001_add_video_fields_postgres.sql

# Rollback migration
psql -U <username> -d <database> -f scripts/database/migrations/002_rollback_video_fields_postgres.sql
```

### MySQL

```bash
# Apply migration
mysql -u <username> -p <database> < scripts/database/migrations/001_add_video_fields_mysql.sql

# Rollback migration
mysql -u <username> -p <database> < scripts/database/migrations/002_rollback_video_fields_mysql.sql
```

## Field Details

### Video Duration

- **video_duration_min**: Minimum acceptable video ad duration in seconds
- **video_duration_max**: Maximum acceptable video ad duration in seconds

**Example:**
```sql
UPDATE stored_imps
SET video_duration_min = 15,
    video_duration_max = 30
WHERE id = 'video-imp-123';
```

**Use Case:** Pre-roll ads typically 15-30 seconds, mid-roll ads 30-60 seconds

### Video Protocols

Array of supported video protocols based on OpenRTB spec:

| Value | Protocol |
|-------|----------|
| 1 | VAST 1.0 |
| 2 | VAST 2.0 |
| 3 | VAST 3.0 |
| 4 | VAST 1.0 Wrapper |
| 5 | VAST 2.0 Wrapper |
| 6 | VAST 3.0 Wrapper |
| 7 | VAST 4.0 |
| 8 | VAST 4.0 Wrapper |
| 9 | DAAST 1.0 |
| 10 | DAAST 1.0 Wrapper |
| 11 | VAST 4.1 |
| 12 | VAST 4.1 Wrapper |

**PostgreSQL Example:**
```sql
UPDATE stored_imps
SET video_protocols = ARRAY[2,3,7]
WHERE id = 'video-imp-123';
```

**MySQL Example:**
```sql
UPDATE stored_imps
SET video_protocols = JSON_ARRAY(2, 3, 7)
WHERE id = 'video-imp-123';
```

### Video Start Delay

Start delay indicates when the video ad plays:

| Value | Placement |
|-------|-----------|
| -1 | Pre-roll (before content) |
| 0 | Mid-roll (during content) |
| >0 | Post-roll (after content, value = seconds into content) |

**Example:**
```sql
-- Pre-roll ad
UPDATE stored_imps SET video_start_delay = -1 WHERE id = 'preroll-imp';

-- Mid-roll ad (during content)
UPDATE stored_imps SET video_start_delay = 0 WHERE id = 'midroll-imp';

-- Post-roll ad (after 300 seconds of content)
UPDATE stored_imps SET video_start_delay = 300 WHERE id = 'postroll-imp';
```

### Video MIME Types

Array of supported video MIME types:

Common values:
- `video/mp4`
- `video/webm`
- `video/ogg`
- `application/javascript` (for VPAID)
- `application/x-shockwave-flash` (for legacy Flash)

**PostgreSQL Example:**
```sql
UPDATE stored_imps
SET video_mimes = ARRAY['video/mp4', 'video/webm']
WHERE id = 'video-imp-123';
```

**MySQL Example:**
```sql
UPDATE stored_imps
SET video_mimes = JSON_ARRAY('video/mp4', 'video/webm')
WHERE id = 'video-imp-123';
```

### Skip Settings

- **video_skippable**: Boolean indicating if the ad can be skipped
- **video_skip_delay**: Seconds before the skip button appears

**Example:**
```sql
-- Skippable after 5 seconds
UPDATE stored_imps
SET video_skippable = TRUE,
    video_skip_delay = 5
WHERE id = 'video-imp-123';

-- Non-skippable ad
UPDATE stored_imps
SET video_skippable = FALSE,
    video_skip_delay = NULL
WHERE id = 'video-imp-456';
```

## Indexes

The migration creates indexes to optimize video-related queries:

### PostgreSQL

- `idx_stored_requests_video_duration` - B-tree index on (video_duration_min, video_duration_max)
- `idx_stored_requests_video_protocols` - GIN index on video_protocols array
- `idx_stored_requests_video_mimes` - GIN index on video_mimes array
- `idx_stored_imps_video_duration` - B-tree index on (video_duration_min, video_duration_max)
- `idx_stored_imps_video_protocols` - GIN index on video_protocols array
- `idx_stored_imps_video_mimes` - GIN index on video_mimes array

### MySQL

- `idx_stored_requests_video_duration` - Index on (video_duration_min, video_duration_max)
- `idx_stored_imps_video_duration` - Index on (video_duration_min, video_duration_max)

## Query Examples

### Find all video impressions with specific duration

**PostgreSQL:**
```sql
SELECT * FROM stored_imps
WHERE video_duration_min >= 15
  AND video_duration_max <= 30;
```

### Find impressions supporting VAST 3.0

**PostgreSQL:**
```sql
SELECT * FROM stored_imps
WHERE video_protocols @> ARRAY[3];
```

**MySQL:**
```sql
SELECT * FROM stored_imps
WHERE JSON_CONTAINS(video_protocols, '3');
```

### Find skippable video ads

```sql
SELECT * FROM stored_imps
WHERE video_skippable = TRUE
  AND video_skip_delay <= 5;
```

### Find pre-roll video impressions

```sql
SELECT * FROM stored_imps
WHERE video_start_delay = -1;
```

## Backwards Compatibility

All video fields are **nullable** and have **default values** to ensure backwards compatibility:

- Existing stored requests/imps will have NULL values for video fields
- The `video_skippable` field defaults to FALSE
- Applications should handle NULL values gracefully

## Integration with OpenRTB

These fields map directly to OpenRTB Video object fields:

| Database Field | OpenRTB Field |
|----------------|---------------|
| video_duration_min | imp.video.minduration |
| video_duration_max | imp.video.maxduration |
| video_protocols | imp.video.protocols |
| video_start_delay | imp.video.startdelay |
| video_mimes | imp.video.mimes |
| video_skip | imp.video.skip |
| video_skipmin | imp.video.skipmin |

## Testing

After running the migration, verify the schema:

### PostgreSQL

```sql
-- Check columns exist
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'stored_imps'
  AND column_name LIKE 'video_%';

-- Check indexes exist
SELECT indexname FROM pg_indexes
WHERE tablename = 'stored_imps'
  AND indexname LIKE '%video%';
```

### MySQL

```sql
-- Check columns exist
DESCRIBE stored_imps;

-- Check indexes exist
SHOW INDEX FROM stored_imps WHERE Key_name LIKE '%video%';
```

## Troubleshooting

### Migration fails with "table does not exist"

Ensure the `stored_requests` and `stored_imps` tables exist in your database. These tables are part of the base Prebid Server schema.

### Migration fails with "column already exists"

The migration uses `IF NOT EXISTS` clauses to prevent errors if columns already exist. If you still encounter errors, check for:
- Partial previous migrations
- Manual schema modifications
- Different column names

### Performance issues after migration

The migration adds indexes to optimize common queries. If you experience performance issues:
1. Run `ANALYZE` (PostgreSQL) or `ANALYZE TABLE` (MySQL) to update statistics
2. Check index usage with `EXPLAIN` queries
3. Consider adding additional indexes for your specific query patterns

## Future Migrations

Future migrations will be numbered sequentially (003, 004, etc.) and will include:
- Additional CTV-specific fields
- VAST event tracking tables
- Device targeting tables
- Analytics tables for video metrics

## Support

For issues or questions about these migrations:
1. Check the [Prebid Server documentation](https://docs.prebid.org/prebid-server/)
2. Review the [OpenRTB specification](https://www.iab.com/guidelines/real-time-bidding-rtb-project/)
3. Open an issue on the [Prebid Server GitHub repository](https://github.com/prebid/prebid-server)
