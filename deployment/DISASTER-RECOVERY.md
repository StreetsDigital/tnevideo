# Disaster Recovery Plan - TNE Catalyst

## Date Created: 2026-01-19

## Overview

This document provides step-by-step procedures for recovering the TNE Catalyst ad exchange from various disaster scenarios.

**Recovery Time Objective (RTO):** < 30 minutes
**Recovery Point Objective (RPO):** < 24 hours (daily backups)
**Backup Schedule:** Daily at 2 AM UTC
**Retention Policy:** 7 daily, 4 weekly, 3 monthly

---

## Table of Contents

1. [Backup System Overview](#backup-system-overview)
2. [Common Recovery Scenarios](#common-recovery-scenarios)
3. [Database Restore Procedures](#database-restore-procedures)
4. [Full System Recovery](#full-system-recovery)
5. [Backup Verification](#backup-verification)
6. [Emergency Contacts](#emergency-contacts)

---

## Backup System Overview

### Automated Backup System

The backup system runs as a Docker container (`catalyst-backup`) with the following features:

- **Automated daily backups** at 2 AM UTC (configurable)
- **Retention policy**: 7 daily, 4 weekly, 3 monthly
- **Backup verification**: Integrity checked after each backup
- **S3 upload**: Optional cloud backup to AWS S3
- **Compression**: PostgreSQL custom format with built-in compression (~85% size reduction)

### Backup Locations

**Local backups:**
```
/backups/
├── latest/       # Most recent backup (always latest.backup)
├── daily/        # Last 7 days
├── weekly/       # Last 4 weeks (Sundays only)
└── monthly/      # Last 3 months (1st of month)
```

**Cloud backups (if S3 configured):**
```
s3://<bucket>/catalyst-backups/
├── latest/latest.backup
└── daily/catalyst_backup_<timestamp>.backup
```

### Backup File Format

- **Format**: PostgreSQL custom format (binary compressed, not gzipped SQL)
- **Filename**: `catalyst_backup_YYYYMMDD_HHMMSS.backup`
- **Example**: `catalyst_backup_20260119_020000.backup`
- **Note**: Use `pg_restore` to restore (not `psql`)

---

## Common Recovery Scenarios

### Scenario 1: Accidental Data Deletion

**Symptoms:**
- Publisher or bidder accidentally deleted
- Auction configuration mistakenly changed
- User reports missing data

**Recovery Steps:**

1. **Identify the timeframe** when data was deleted
2. **Find appropriate backup** from before deletion:
   ```bash
   docker exec -it catalyst-backup ls -lh /backups/daily/
   ```

3. **Stop the application** to prevent further changes:
   ```bash
   docker stop catalyst
   ```

4. **Restore from backup** (see [Database Restore Procedures](#database-restore-procedures))

5. **Restart application:**
   ```bash
   docker start catalyst
   ```

6. **Verify data** is restored

**Estimated Recovery Time:** 10-15 minutes

---

### Scenario 2: Database Corruption

**Symptoms:**
- PostgreSQL crashes repeatedly
- `pg_dump` fails with errors
- Inconsistent query results
- Database refuses connections

**Recovery Steps:**

1. **Attempt database repair** first:
   ```bash
   docker exec -it catalyst-postgres psql -U catalyst -d catalyst -c "REINDEX DATABASE catalyst;"
   ```

2. If repair fails, **restore from latest backup**:
   ```bash
   # Use latest backup
   docker exec -it catalyst-backup \
     bash -c "FORCE=true /usr/local/bin/restore-postgres.sh /backups/latest/latest.backup"
   ```

3. **Verify database integrity**:
   ```bash
   docker exec -it catalyst-postgres psql -U catalyst -d catalyst -c "\dt"
   ```

4. **Restart all services**:
   ```bash
   docker-compose restart
   ```

**Estimated Recovery Time:** 15-20 minutes

---

### Scenario 3: Complete Server Failure

**Symptoms:**
- Server hardware failure
- Cloud instance terminated
- Operating system corruption
- Complete data center outage

**Recovery Steps:**

1. **Provision new server** with Docker installed

2. **Clone repository:**
   ```bash
   git clone https://github.com/thenexusengine/tne_springwire.git
   cd tne_springwire/deployment
   ```

3. **Restore environment configuration:**
   ```bash
   # Copy .env file from backup or secrets manager
   cp .env.production .env
   ```

4. **Download latest backup from S3** (if cloud backups enabled):
   ```bash
   aws s3 cp s3://<bucket>/catalyst-backups/latest/latest.backup /tmp/backup.backup
   ```

5. **Start infrastructure:**
   ```bash
   docker-compose up -d postgres redis
   ```

6. **Wait for PostgreSQL to be ready:**
   ```bash
   docker-compose logs -f postgres
   # Wait for "database system is ready to accept connections"
   ```

7. **Restore database:**
   ```bash
   docker run --rm \
     --network catalyst-network \
     -v /tmp/backup.backup:/backup.backup \
     -e POSTGRES_HOST=catalyst-postgres \
     -e POSTGRES_DB=catalyst \
     -e POSTGRES_USER=catalyst \
     -e POSTGRES_PASSWORD=$DB_PASSWORD \
     -e FORCE=true \
     postgres:16-alpine \
     sh -c "pg_restore -h catalyst-postgres -U catalyst -d catalyst --no-owner --no-acl /backup.backup"
   ```

8. **Start all services:**
   ```bash
   docker-compose up -d
   ```

9. **Verify system health:**
   ```bash
   curl http://localhost:8000/health/ready
   ```

**Estimated Recovery Time:** 30-45 minutes

---

### Scenario 4: Partial Data Loss (Redis Cache)

**Symptoms:**
- Redis container crashed
- Cache data lost
- Performance degradation

**Recovery Steps:**

Redis is a cache only - no restore needed. The system will rebuild cache from PostgreSQL.

1. **Restart Redis:**
   ```bash
   docker-compose restart redis
   ```

2. **Monitor cache rebuild:**
   ```bash
   docker-compose logs -f redis
   ```

**Estimated Recovery Time:** < 5 minutes

---

## Database Restore Procedures

### Method 1: Using the Restore Script (Recommended)

**Prerequisites:**
- Backup service container running
- Backup file available in `/backups/`

**Steps:**

1. **List available backups:**
   ```bash
   docker exec -it catalyst-backup /usr/local/bin/restore-postgres.sh
   ```

2. **Restore from latest:**
   ```bash
   docker exec -it catalyst-backup \
     bash -c "FORCE=true /usr/local/bin/restore-postgres.sh /backups/latest/latest.backup"
   ```

3. **Restore from specific date:**
   ```bash
   docker exec -it catalyst-backup \
     bash -c "FORCE=true /usr/local/bin/restore-postgres.sh /backups/daily/daily_20260119.backup"
   ```

### Method 2: Manual Restore

**Steps:**

1. **Stop application:**
   ```bash
   docker stop catalyst
   ```

2. **Copy backup file to PostgreSQL container:**
   ```bash
   docker cp /path/to/backup.backup catalyst-postgres:/tmp/backup.backup
   ```

3. **Drop and recreate database:**
   ```bash
   docker exec -it catalyst-postgres psql -U catalyst -d postgres <<EOF
   SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'catalyst';
   DROP DATABASE IF EXISTS catalyst;
   CREATE DATABASE catalyst OWNER catalyst;
   EOF
   ```

4. **Restore backup:**
   ```bash
   docker exec -it catalyst-postgres \
     pg_restore -U catalyst -d catalyst --no-owner --no-acl /tmp/backup.backup
   ```

5. **Restart application:**
   ```bash
   docker start catalyst
   ```

### Method 3: Restore from S3

**Prerequisites:**
- AWS CLI configured with proper credentials
- S3 bucket with backups

**Steps:**

1. **Download from S3:**
   ```bash
   aws s3 cp s3://<bucket>/catalyst-backups/latest/latest.backup /tmp/backup.backup
   ```

2. **Follow Method 2** steps 2-5 above

---

## Full System Recovery

### Complete Deployment Recovery (Nuclear Option)

Use this when all else fails. This will rebuild the entire system from scratch.

**Steps:**

1. **Backup current state** (if possible):
   ```bash
   docker exec catalyst-backup /usr/local/bin/backup-postgres.sh
   docker cp catalyst-backup:/backups/latest/latest.backup /safe/location/
   ```

2. **Stop and remove all containers:**
   ```bash
   docker-compose down -v
   ```

3. **Remove all volumes** (⚠️ DESTRUCTIVE):
   ```bash
   docker volume rm deployment_postgres-data deployment_redis-data
   ```

4. **Pull latest code:**
   ```bash
   git pull origin master
   ```

5. **Rebuild containers:**
   ```bash
   docker-compose build --no-cache
   ```

6. **Start infrastructure:**
   ```bash
   docker-compose up -d postgres redis
   ```

7. **Wait for PostgreSQL ready:**
   ```bash
   until docker exec catalyst-postgres pg_isready -U catalyst; do
     echo "Waiting for PostgreSQL..."
     sleep 2
   done
   ```

8. **Restore database:**
   ```bash
   docker exec -it catalyst-postgres \
     pg_restore -U catalyst -d catalyst --clean --if-exists /backups/latest/latest.backup
   ```

9. **Start all services:**
   ```bash
   docker-compose up -d
   ```

10. **Verify health:**
    ```bash
    curl http://localhost:8000/health/ready
    ```

**Estimated Recovery Time:** 45-60 minutes

---

## Backup Verification

### Regular Backup Testing (Monthly Recommended)

**Test restore to separate environment:**

1. **Spin up test environment:**
   ```bash
   docker run -d --name test-postgres \
     -e POSTGRES_DB=test_restore \
     -e POSTGRES_USER=catalyst \
     -e POSTGRES_PASSWORD=changeme \
     postgres:16-alpine
   ```

2. **Restore latest backup:**
   ```bash
   docker exec -i test-postgres pg_restore \
     -U catalyst -d test_restore --no-owner --no-acl \
     < /backups/latest/latest.backup
   ```

3. **Verify data:**
   ```bash
   docker exec -it test-postgres psql -U catalyst -d test_restore -c "
     SELECT COUNT(*) FROM publishers;
     SELECT COUNT(*) FROM bidders;
   "
   ```

4. **Clean up:**
   ```bash
   docker stop test-postgres && docker rm test-postgres
   ```

### Automated Verification

Backup integrity is automatically verified after each backup:
- File size > 0 check
- PostgreSQL custom format validation

View verification logs:
```bash
docker logs catalyst-backup | grep -E "(Backup completed|failed)"
```

---

## Emergency Runbook

### Quick Reference

| Issue | Command |
|-------|---------|
| View latest backup | `docker exec catalyst-backup ls -lh /backups/latest/` |
| Manual backup now | `docker exec catalyst-backup /usr/local/bin/backup-postgres.sh` |
| Restore latest | `docker exec catalyst-backup bash -c "FORCE=true /usr/local/bin/restore-postgres.sh /backups/latest/latest.backup"` |
| Check backup logs | `docker logs catalyst-backup` |
| List all backups | `docker exec catalyst-backup find /backups -name "*.backup" -ls` |
| Download from S3 | `aws s3 cp s3://$BACKUP_S3_BUCKET/catalyst-backups/latest/latest.backup .` |

### Pre-Restore Checklist

- [ ] Identify which backup to restore from
- [ ] Verify backup file exists and is not corrupted
- [ ] Stop application to prevent data changes
- [ ] Document current state (table counts, etc.)
- [ ] Have rollback plan ready
- [ ] Notify team of maintenance window

### Post-Restore Checklist

- [ ] Verify table counts match expectations
- [ ] Check critical publisher/bidder records exist
- [ ] Test API endpoints (`/health/ready`)
- [ ] Review application logs for errors
- [ ] Run smoke tests (sample auction request)
- [ ] Notify team of completion

---

## Monitoring and Alerts

### Backup Health Checks

**Monitor backup success:**
```bash
# Check if backup ran today
docker exec catalyst-backup find /backups/daily -name "daily_$(date +%Y%m%d).backup"
```

**Set up alerts for:**
- Backup failures (no backup file created)
- Backup size anomalies (too small/large)
- S3 upload failures
- Disk space on backup volume

### Recommended Alerts

```yaml
# Example Prometheus alert rules
groups:
  - name: backup_alerts
    rules:
      - alert: BackupMissing
        expr: time() - backup_last_success_timestamp > 86400
        annotations:
          summary: "No backup in last 24 hours"

      - alert: BackupVolumeFull
        expr: backup_volume_used_percent > 90
        annotations:
          summary: "Backup volume >90% full"
```

---

## Environment-Specific Notes

### Production

- **Backup schedule:** Daily at 2 AM UTC
- **S3 bucket:** `s3://catalyst-prod-backups`
- **Retention:** 7 daily, 4 weekly, 3 monthly
- **Encryption:** S3 server-side encryption enabled

### Staging

- **Backup schedule:** Daily at 3 AM UTC
- **S3 bucket:** `s3://catalyst-staging-backups`
- **Retention:** 3 daily, 2 weekly
- **Note:** Can restore from production backups for testing

---

## Emergency Contacts

| Role | Contact | Escalation |
|------|---------|------------|
| Database Admin | TBD | TBD |
| DevOps Lead | TBD | TBD |
| On-Call Engineer | TBD | PagerDuty |
| AWS Support | Support case | Critical ticket |

---

## Testing Schedule

- **Monthly:** Test restore to isolated environment
- **Quarterly:** Full disaster recovery drill
- **Annually:** Multi-region failover test

---

## Revision History

| Date | Version | Changes |
|------|---------|---------|
| 2026-01-19 | 1.0 | Initial disaster recovery plan created |

---

**Last Updated:** 2026-01-19
**Next Review:** 2026-04-19 (Quarterly)
