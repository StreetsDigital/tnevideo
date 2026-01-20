#!/bin/bash
# Emergency Rollback Script
# Quickly reverts to previous deployment version

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
BACKUP_TAG="${1:-}"
FORCE="${FORCE:-false}"

echo "========================================="
echo "🔄 Emergency Rollback Script"
echo "========================================="
echo ""

# Function to get current image tag
get_current_tag() {
    docker inspect catalyst --format='{{.Config.Image}}' 2>/dev/null | cut -d: -f2 || echo "unknown"
}

# Function to list available backups
list_backups() {
    echo "Available database backups:"
    echo ""
    if [ -d "/backups" ]; then
        echo "Latest:"
        ls -lht /backups/latest/*.sql.gz 2>/dev/null || echo "  (none)"
        echo ""
        echo "Daily (last 7):"
        ls -lht /backups/daily/*.sql.gz 2>/dev/null | head -7 || echo "  (none)"
    else
        echo "  Run inside backup container or mount /backups volume"
    fi
}

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ docker-compose not found${NC}"
    exit 1
fi

CURRENT_TAG=$(get_current_tag)
echo "Current deployment: ${CURRENT_TAG}"
echo ""

# If no backup specified, show options
if [ -z "$BACKUP_TAG" ]; then
    echo -e "${YELLOW}⚠️  No rollback target specified${NC}"
    echo ""
    echo "Usage: $0 <backup-date> [FORCE=true]"
    echo ""
    echo "Examples:"
    echo "  $0 20260119              # Rollback to January 19, 2026 backup"
    echo "  $0 latest                # Rollback to latest backup"
    echo "  FORCE=true $0 20260119   # Force rollback without confirmation"
    echo ""
    list_backups
    exit 1
fi

# Determine backup file
if [ "$BACKUP_TAG" = "latest" ]; then
    BACKUP_FILE="/backups/latest/latest.sql.gz"
else
    BACKUP_FILE="/backups/daily/daily_${BACKUP_TAG}.sql.gz"
fi

echo "Rollback target: ${BACKUP_TAG}"
echo "Backup file: ${BACKUP_FILE}"
echo ""

# Confirmation
if [ "$FORCE" != "true" ]; then
    echo -e "${RED}⚠️  WARNING: This will:${NC}"
    echo "  1. Stop the current deployment"
    echo "  2. Restore database from backup: ${BACKUP_FILE}"
    echo "  3. Restart services"
    echo ""
    echo "Current data will be OVERWRITTEN!"
    echo ""
    read -p "Continue with rollback? (type 'yes' to confirm): " confirm

    if [ "$confirm" != "yes" ]; then
        echo "Rollback cancelled."
        exit 0
    fi
fi

echo ""
echo "========================================="
echo "Step 1: Creating Emergency Backup"
echo "========================================="

# Create emergency backup before rollback
echo "Creating pre-rollback backup..."
EMERGENCY_BACKUP="/backups/emergency_rollback_$(date +%Y%m%d_%H%M%S).sql.gz"

if docker exec catalyst-backup /usr/local/bin/backup-postgres.sh; then
    docker exec catalyst-backup bash -c "cp /backups/latest/latest.sql.gz ${EMERGENCY_BACKUP}"
    echo -e "${GREEN}✅ Emergency backup created: ${EMERGENCY_BACKUP}${NC}"
else
    echo -e "${RED}❌ Emergency backup failed${NC}"
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo ""
echo "========================================="
echo "Step 2: Stopping Services"
echo "========================================="

docker-compose stop catalyst
echo -e "${GREEN}✅ Services stopped${NC}"

echo ""
echo "========================================="
echo "Step 3: Restoring Database"
echo "========================================="

# Restore database
if docker exec catalyst-backup bash -c "FORCE=true /usr/local/bin/restore-postgres.sh ${BACKUP_FILE}"; then
    echo -e "${GREEN}✅ Database restored from backup${NC}"
else
    echo -e "${RED}❌ Database restore failed!${NC}"
    echo ""
    echo "Attempting to restore from emergency backup..."
    if docker exec catalyst-backup bash -c "FORCE=true /usr/local/bin/restore-postgres.sh ${EMERGENCY_BACKUP}"; then
        echo -e "${GREEN}✅ Restored from emergency backup${NC}"
        echo -e "${YELLOW}⚠️  System is back to pre-rollback state${NC}"
    else
        echo -e "${RED}❌ Emergency restore also failed!${NC}"
        echo "Manual intervention required."
    fi
    exit 1
fi

echo ""
echo "========================================="
echo "Step 4: Restarting Services"
echo "========================================="

docker-compose start catalyst

# Wait for health check
echo "Waiting for service to become healthy..."
ATTEMPTS=0
MAX_ATTEMPTS=30

until curl -sf http://localhost:8000/health/ready > /dev/null 2>&1; do
    ATTEMPTS=$((ATTEMPTS + 1))
    if [ $ATTEMPTS -ge $MAX_ATTEMPTS ]; then
        echo -e "${RED}❌ Service failed to become healthy after ${MAX_ATTEMPTS} attempts${NC}"
        echo "Check logs: docker-compose logs catalyst"
        exit 1
    fi
    echo -n "."
    sleep 2
done

echo ""
echo -e "${GREEN}✅ Service is healthy${NC}"

echo ""
echo "========================================="
echo "Step 5: Verification"
echo "========================================="

# Run smoke tests if available
if [ -f "./smoke-tests.sh" ]; then
    echo "Running smoke tests..."
    if ./smoke-tests.sh; then
        echo -e "${GREEN}✅ Smoke tests passed${NC}"
    else
        echo -e "${RED}❌ Smoke tests failed${NC}"
        echo "Service may not be functioning correctly"
    fi
else
    # Manual checks
    echo "Checking endpoints..."

    if curl -sf http://localhost:8000/health > /dev/null; then
        echo -e "${GREEN}✅${NC} /health"
    else
        echo -e "${RED}❌${NC} /health"
    fi

    if curl -sf http://localhost:8000/health/ready > /dev/null; then
        echo -e "${GREEN}✅${NC} /health/ready"
    else
        echo -e "${RED}❌${NC} /health/ready"
    fi

    if curl -sf http://localhost:8000/metrics > /dev/null; then
        echo -e "${GREEN}✅${NC} /metrics"
    else
        echo -e "${RED}❌${NC} /metrics"
    fi
fi

# Check logs for errors
echo ""
echo "Recent error logs:"
docker-compose logs --tail=20 catalyst | grep -i error || echo "  (no errors found)"

echo ""
echo "========================================="
echo "Rollback Complete"
echo "========================================="
echo ""
echo -e "${GREEN}✅ Rollback successful${NC}"
echo ""
echo "Summary:"
echo "  • Restored from: ${BACKUP_FILE}"
echo "  • Emergency backup: ${EMERGENCY_BACKUP}"
echo "  • Current tag: $(get_current_tag)"
echo "  • Services: Running"
echo "  • Health: $(curl -s http://localhost:8000/health | jq -r '.status' 2>/dev/null || echo 'unknown')"
echo ""
echo "Next steps:"
echo "  1. Monitor logs: docker-compose logs -f catalyst"
echo "  2. Check metrics in Grafana"
echo "  3. Verify functionality with real traffic"
echo "  4. Investigate root cause of issue"
echo ""
echo "To rollback the rollback (restore current state):"
echo "  FORCE=true $0 emergency_rollback_$(date +%Y%m%d)_*"
echo ""
