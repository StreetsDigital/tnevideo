# Publisher Configuration Guide

**How to configure publishers in TNE Catalyst**

Catalyst supports two methods for publisher configuration:
1. **Environment Variables** - Simple, requires restart
2. **Redis** - Dynamic, no restart needed (recommended for production)

---

## Understanding Publisher Configuration

### What Gets Configured Per Publisher?

1. **Publisher ID** - Unique identifier sent in bid requests
2. **Allowed Domains** - Which domains/apps this publisher can use
3. **Rate Limits** - Per-publisher request limits
4. **IVT Monitoring** - Invalid traffic detection

### Where Publishers Are Defined

Publishers can be defined in **3 places** (checked in this order):

1. **Redis** (priority) - Dynamic, persistent
2. **Environment Variables** - Static, requires restart
3. **Allow Unregistered** - Development mode fallback

---

## Method 1: Environment Variables (Simple)

**Best for**: Development, small deployments, static publisher lists

### Configuration in .env files

Add to `deployment/.env.production`:

```bash
# Publisher Authentication
PUBLISHER_AUTH_ENABLED=true
PUBLISHER_ALLOW_UNREGISTERED=false
PUBLISHER_VALIDATE_DOMAIN=true

# Registered Publishers (format: pubID:domain1|domain2,pubID2:domain)
REGISTERED_PUBLISHERS=pub123:example.com|*.example.com,pub456:anothersite.com
```

### Format Explained

```bash
REGISTERED_PUBLISHERS=pubID:domains,pubID:domains

# Examples:
# Single domain:
REGISTERED_PUBLISHERS=pub123:example.com

# Multiple domains (pipe-separated):
REGISTERED_PUBLISHERS=pub123:example.com|cdn.example.com

# Wildcard subdomains:
REGISTERED_PUBLISHERS=pub123:*.example.com

# Allow any domain:
REGISTERED_PUBLISHERS=pub123:*

# Multiple publishers (comma-separated):
REGISTERED_PUBLISHERS=pub123:example.com,pub456:another.com,pub789:*.site3.com
```

### Restart Required

After changing environment variables:

```bash
cd /opt/catalyst
docker compose restart catalyst
```

---

## Method 2: Redis (Dynamic - Recommended)

**Best for**: Production, frequent changes, multiple publishers

### Why Redis?

✅ **No restart required** - Changes apply immediately
✅ **Dynamic management** - Add/remove publishers on the fly
✅ **Centralized** - Single source of truth
✅ **Scalable** - Handles hundreds of publishers easily

### Redis Key Structure

```
Key: tne_catalyst:publishers
Type: Hash
Format:
  Field: publisher_id
  Value: allowed_domains (pipe-separated, or "*" for any)
```

### Adding Publishers via Redis CLI

**Method 1: Redis CLI directly**

```bash
# Connect to Redis container
docker exec -it catalyst-redis redis-cli

# Add a single publisher
HSET tne_catalyst:publishers pub123 "example.com|*.example.com"

# Add multiple publishers
HSET tne_catalyst:publishers pub456 "anothersite.com"
HSET tne_catalyst:publishers pub789 "*"  # Allow any domain

# View all publishers
HGETALL tne_catalyst:publishers

# Check specific publisher
HGET tne_catalyst:publishers pub123

# Remove a publisher
HDEL tne_catalyst:publishers pub123

# Exit
exit
```

**Method 2: Using redis-cli remotely**

```bash
# If Redis password is set
docker exec -it catalyst-redis redis-cli -a YOUR_REDIS_PASSWORD

# Then run HSET commands as above
```

---

## Method 3: Management Script (Easiest)

I'll create a helper script for you:

**File**: `deployment/manage-publishers.sh`

```bash
#!/bin/bash
# Publisher Management Script for Catalyst

REDIS_CONTAINER="catalyst-redis"
REDIS_KEY="tne_catalyst:publishers"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Redis is running
check_redis() {
    if ! docker ps | grep -q $REDIS_CONTAINER; then
        echo -e "${RED}Error: Redis container not running${NC}"
        exit 1
    fi
}

# List all publishers
list_publishers() {
    echo -e "${GREEN}Registered Publishers:${NC}"
    docker exec -it $REDIS_CONTAINER redis-cli HGETALL $REDIS_KEY | \
        awk 'NR%2==1{printf "%-20s", $0} NR%2==0{print " -> " $0}'
}

# Add publisher
add_publisher() {
    local pub_id=$1
    local domains=$2

    if [ -z "$pub_id" ] || [ -z "$domains" ]; then
        echo -e "${RED}Usage: $0 add <publisher_id> <domains>${NC}"
        echo "Example: $0 add pub123 'example.com|*.example.com'"
        exit 1
    fi

    docker exec -it $REDIS_CONTAINER redis-cli HSET $REDIS_KEY "$pub_id" "$domains"
    echo -e "${GREEN}✓ Added publisher: $pub_id -> $domains${NC}"
}

# Remove publisher
remove_publisher() {
    local pub_id=$1

    if [ -z "$pub_id" ]; then
        echo -e "${RED}Usage: $0 remove <publisher_id>${NC}"
        exit 1
    fi

    docker exec -it $REDIS_CONTAINER redis-cli HDEL $REDIS_KEY "$pub_id"
    echo -e "${GREEN}✓ Removed publisher: $pub_id${NC}"
}

# Check publisher
check_publisher() {
    local pub_id=$1

    if [ -z "$pub_id" ]; then
        echo -e "${RED}Usage: $0 check <publisher_id>${NC}"
        exit 1
    fi

    local domains=$(docker exec -it $REDIS_CONTAINER redis-cli HGET $REDIS_KEY "$pub_id")
    if [ -z "$domains" ]; then
        echo -e "${YELLOW}Publisher $pub_id not found${NC}"
    else
        echo -e "${GREEN}Publisher: $pub_id${NC}"
        echo -e "${GREEN}Domains: $domains${NC}"
    fi
}

# Update publisher domains
update_publisher() {
    local pub_id=$1
    local domains=$2

    if [ -z "$pub_id" ] || [ -z "$domains" ]; then
        echo -e "${RED}Usage: $0 update <publisher_id> <new_domains>${NC}"
        exit 1
    fi

    docker exec -it $REDIS_CONTAINER redis-cli HSET $REDIS_KEY "$pub_id" "$domains"
    echo -e "${GREEN}✓ Updated publisher: $pub_id -> $domains${NC}"
}

# Main
check_redis

case "$1" in
    list|ls)
        list_publishers
        ;;
    add)
        add_publisher "$2" "$3"
        ;;
    remove|rm)
        remove_publisher "$2"
        ;;
    check)
        check_publisher "$2"
        ;;
    update)
        update_publisher "$2" "$3"
        ;;
    *)
        echo "Catalyst Publisher Management"
        echo ""
        echo "Usage: $0 <command> [options]"
        echo ""
        echo "Commands:"
        echo "  list, ls                    List all registered publishers"
        echo "  add <id> <domains>          Add new publisher"
        echo "  remove, rm <id>             Remove publisher"
        echo "  check <id>                  Check specific publisher"
        echo "  update <id> <domains>       Update publisher domains"
        echo ""
        echo "Domain Format:"
        echo "  Single domain:      example.com"
        echo "  Multiple domains:   example.com|cdn.example.com"
        echo "  Wildcard subdomain: *.example.com"
        echo "  Allow any domain:   *"
        echo ""
        echo "Examples:"
        echo "  $0 list"
        echo "  $0 add pub123 'example.com'"
        echo "  $0 add pub456 'example.com|*.example.com'"
        echo "  $0 check pub123"
        echo "  $0 update pub123 'newdomain.com|*.newdomain.com'"
        echo "  $0 remove pub123"
        ;;
esac
```

---

## Quick Start: Adding Your First Publisher

### Step 1: Choose Method

**For production** - Use Redis (recommended):

```bash
# Copy and make executable
cd /opt/catalyst
chmod +x manage-publishers.sh

# Add your first publisher
./manage-publishers.sh add pub123 "yourpublisher.com|*.yourpublisher.com"

# List all publishers
./manage-publishers.sh list
```

**For development** - Use environment variables:

```bash
# Edit .env file
nano .env

# Add line:
REGISTERED_PUBLISHERS=pub123:yourpublisher.com|*.yourpublisher.com

# Restart
docker compose restart catalyst
```

### Step 2: Configure CORS

Publishers must also be in CORS allowlist:

```bash
# In .env file
CORS_ALLOWED_ORIGINS=https://yourpublisher.com,https://*.yourpublisher.com
```

### Step 3: Test

Send a test bid request with publisher ID:

```json
{
  "id": "test-auction",
  "site": {
    "domain": "yourpublisher.com",
    "publisher": {
      "id": "pub123"
    }
  },
  "imp": [...]
}
```

---

## Common Scenarios

### Scenario 1: Add New Publisher

```bash
# Via script (recommended)
./manage-publishers.sh add pub_newsite "newsite.com|*.newsite.com"

# Via Redis directly
docker exec -it catalyst-redis redis-cli HSET tne_catalyst:publishers pub_newsite "newsite.com|*.newsite.com"
```

### Scenario 2: Publisher Changes Domains

```bash
# Update domains
./manage-publishers.sh update pub123 "newdomain.com|old-domain.com"

# Or via Redis
docker exec -it catalyst-redis redis-cli HSET tne_catalyst:publishers pub123 "newdomain.com|old-domain.com"
```

### Scenario 3: Remove Publisher

```bash
# Via script
./manage-publishers.sh remove pub_oldsite

# Via Redis
docker exec -it catalyst-redis redis-cli HDEL tne_catalyst:publishers pub_oldsite
```

### Scenario 4: Publisher with Multiple Subdomains

```bash
# Wildcard subdomains
./manage-publishers.sh add pub123 "*.example.com"

# Or specific subdomains
./manage-publishers.sh add pub123 "www.example.com|blog.example.com|shop.example.com"
```

### Scenario 5: Testing Publisher (Allow Any Domain)

```bash
# During testing only - allows any domain
./manage-publishers.sh add pub_test "*"

# Remember to restrict later!
./manage-publishers.sh update pub_test "actual-domain.com"
```

---

## Configuration Options

All available environment variables:

```bash
# Publisher Authentication
PUBLISHER_AUTH_ENABLED=true              # Enable publisher validation (default: true)
PUBLISHER_ALLOW_UNREGISTERED=false       # Allow requests without publisher ID (default: false)
PUBLISHER_VALIDATE_DOMAIN=true           # Validate domains match (default: false)
PUBLISHER_AUTH_USE_REDIS=true            # Use Redis for pub config (default: true)

# Registered Publishers (environment method)
REGISTERED_PUBLISHERS=pub1:domain1.com,pub2:domain2.com

# Rate Limiting (per publisher)
# Configured in code, default 100 RPS per publisher
```

---

## Verification

### Check Publisher is Registered

```bash
# Via script
./manage-publishers.sh check pub123

# Via Redis
docker exec -it catalyst-redis redis-cli HGET tne_catalyst:publishers pub123
```

### Test Auction Request

```bash
curl -X POST https://catalyst.springwire.ai/openrtb2/auction \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-123",
    "site": {
      "domain": "yourpublisher.com",
      "publisher": {
        "id": "pub123"
      }
    },
    "imp": [{
      "id": "1",
      "banner": {
        "w": 300,
        "h": 250
      }
    }]
  }'
```

### Check Logs

```bash
# Watch for publisher validation
docker compose logs -f catalyst | grep -i publisher

# Should see:
# "Publisher validation passed" or
# "Publisher not registered" (if misconfigured)
```

---

## Troubleshooting

### Publisher Rejected: "publisher not registered"

**Causes**:
1. Publisher ID not in Redis or environment variables
2. Typo in publisher ID
3. Redis not connected

**Solutions**:
```bash
# Check if publisher exists
./manage-publishers.sh check pub123

# Add if missing
./manage-publishers.sh add pub123 "domain.com"

# Verify Redis connection
docker compose logs catalyst | grep -i redis
```

### Publisher Rejected: "domain not allowed"

**Causes**:
1. Domain validation enabled but domain doesn't match
2. Wrong domain format (missing wildcard)

**Solutions**:
```bash
# Check current domains
./manage-publishers.sh check pub123

# Update with wildcard
./manage-publishers.sh update pub123 "*.example.com|example.com"

# Or disable domain validation (development only)
# In .env: PUBLISHER_VALIDATE_DOMAIN=false
```

### Changes Not Taking Effect

**Environment Variables**:
```bash
# Must restart after .env changes
docker compose restart catalyst
```

**Redis**:
```bash
# Should be immediate, check connection
docker compose logs catalyst | grep -i redis

# Verify publisher exists
docker exec -it catalyst-redis redis-cli HGETALL tne_catalyst:publishers
```

---

## Security Best Practices

### 1. Never Allow Unregistered in Production

```bash
# .env.production
PUBLISHER_AUTH_ENABLED=true
PUBLISHER_ALLOW_UNREGISTERED=false  # ← CRITICAL
```

### 2. Use Domain Validation

```bash
# Validate domains match registered publishers
PUBLISHER_VALIDATE_DOMAIN=true
```

### 3. Use Specific Domains (Not Wildcards)

```bash
# ✅ Good - Specific domains
./manage-publishers.sh add pub123 "www.example.com|blog.example.com"

# ⚠️ Acceptable - Subdomain wildcard
./manage-publishers.sh add pub123 "*.example.com"

# ❌ Bad - Too permissive
./manage-publishers.sh add pub123 "*"
```

### 4. Monitor Publisher Activity

```bash
# Check logs for suspicious activity
docker compose logs catalyst | grep -i "publisher_id"

# Watch for rate limit violations
docker compose logs catalyst | grep -i "rate limit"
```

### 5. Protect Redis

```bash
# In .env.production
REDIS_PASSWORD=strong_password_here  # Never leave empty in production
```

---

## Migration Guide

### Moving from Environment Variables to Redis

```bash
# 1. Export current publishers from .env
# If you have: REGISTERED_PUBLISHERS=pub1:domain1.com,pub2:domain2.com

# 2. Add each to Redis
./manage-publishers.sh add pub1 "domain1.com"
./manage-publishers.sh add pub2 "domain2.com"

# 3. Verify
./manage-publishers.sh list

# 4. Remove from .env (optional, Redis takes priority anyway)
nano .env
# Comment out: # REGISTERED_PUBLISHERS=...

# 5. No restart needed!
```

---

## Summary

| Method | Pros | Cons | Best For |
|--------|------|------|----------|
| **Redis** | Dynamic, no restart, scalable | Requires Redis connection | Production, multiple publishers |
| **Environment Variables** | Simple, version controlled | Requires restart, not dynamic | Development, static configs |
| **Management Script** | Easy to use, safe | Requires shell access | Day-to-day operations |

**Recommended Approach**:
- **Production**: Use Redis with management script
- **Development**: Use environment variables for simplicity

---

**Next Steps**:
1. Install management script: Copy to `/opt/catalyst/manage-publishers.sh`
2. Add your first publisher: `./manage-publishers.sh add pub123 "domain.com"`
3. Configure CORS: Update `CORS_ALLOWED_ORIGINS` in `.env`
4. Test auction: Send test bid request with publisher ID

---

**Last Updated**: 2026-01-13
**Version**: 1.0.0
