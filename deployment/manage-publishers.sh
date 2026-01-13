#!/bin/bash
# Publisher Management Script for Catalyst
# Usage: ./manage-publishers.sh <command> [options]

REDIS_CONTAINER="catalyst-redis"
REDIS_KEY="tne_catalyst:publishers"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if Redis is running
check_redis() {
    if ! docker ps | grep -q $REDIS_CONTAINER; then
        echo -e "${RED}Error: Redis container '$REDIS_CONTAINER' not running${NC}"
        echo -e "${YELLOW}Start with: docker compose up -d${NC}"
        exit 1
    fi
}

# List all publishers
list_publishers() {
    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}Registered Publishers in Catalyst${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"

    local output=$(docker exec $REDIS_CONTAINER redis-cli HGETALL $REDIS_KEY 2>/dev/null)

    if [ -z "$output" ]; then
        echo -e "${YELLOW}No publishers registered yet${NC}"
        echo ""
        echo "Add your first publisher:"
        echo "  $0 add pub123 'example.com'"
        return
    fi

    echo "$output" | awk '
        NR%2==1 { pub=$0 }
        NR%2==0 {
            printf "  %-25s → %s\n", pub, $0
        }'

    local count=$(echo "$output" | wc -l)
    count=$((count / 2))
    echo ""
    echo -e "${GREEN}Total: $count publisher(s)${NC}"
}

# Add publisher
add_publisher() {
    local pub_id=$1
    local domains=$2

    if [ -z "$pub_id" ] || [ -z "$domains" ]; then
        echo -e "${RED}Error: Missing arguments${NC}"
        echo ""
        echo "Usage: $0 add <publisher_id> <domains>"
        echo ""
        echo "Examples:"
        echo "  $0 add pub123 'example.com'"
        echo "  $0 add pub456 'example.com|*.example.com'"
        echo "  $0 add pub789 '*'  # Allow any domain (testing only)"
        exit 1
    fi

    # Check if already exists
    local existing=$(docker exec $REDIS_CONTAINER redis-cli HGET $REDIS_KEY "$pub_id" 2>/dev/null)
    if [ -n "$existing" ]; then
        echo -e "${YELLOW}Warning: Publisher '$pub_id' already exists with domains: $existing${NC}"
        echo -e "${YELLOW}Use 'update' command to change domains${NC}"
        exit 1
    fi

    docker exec $REDIS_CONTAINER redis-cli HSET $REDIS_KEY "$pub_id" "$domains" > /dev/null
    echo -e "${GREEN}✓ Successfully added publisher${NC}"
    echo ""
    echo -e "  Publisher ID: ${BLUE}$pub_id${NC}"
    echo -e "  Allowed Domains: ${BLUE}$domains${NC}"
    echo ""
    echo -e "${YELLOW}Remember to also configure CORS:${NC}"
    echo "  CORS_ALLOWED_ORIGINS=https://yourdomain.com"
}

# Remove publisher
remove_publisher() {
    local pub_id=$1

    if [ -z "$pub_id" ]; then
        echo -e "${RED}Error: Missing publisher ID${NC}"
        echo ""
        echo "Usage: $0 remove <publisher_id>"
        echo ""
        echo "Example:"
        echo "  $0 remove pub123"
        exit 1
    fi

    # Check if exists
    local existing=$(docker exec $REDIS_CONTAINER redis-cli HGET $REDIS_KEY "$pub_id" 2>/dev/null)
    if [ -z "$existing" ]; then
        echo -e "${YELLOW}Publisher '$pub_id' not found${NC}"
        exit 1
    fi

    docker exec $REDIS_CONTAINER redis-cli HDEL $REDIS_KEY "$pub_id" > /dev/null
    echo -e "${GREEN}✓ Successfully removed publisher: $pub_id${NC}"
    echo -e "${YELLOW}Previous domains were: $existing${NC}"
}

# Check publisher
check_publisher() {
    local pub_id=$1

    if [ -z "$pub_id" ]; then
        echo -e "${RED}Error: Missing publisher ID${NC}"
        echo ""
        echo "Usage: $0 check <publisher_id>"
        echo ""
        echo "Example:"
        echo "  $0 check pub123"
        exit 1
    fi

    local domains=$(docker exec $REDIS_CONTAINER redis-cli HGET $REDIS_KEY "$pub_id" 2>/dev/null)

    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
    if [ -z "$domains" ]; then
        echo -e "${YELLOW}Publisher '$pub_id' NOT REGISTERED${NC}"
        echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
        echo ""
        echo "Add this publisher:"
        echo "  $0 add $pub_id 'example.com'"
    else
        echo -e "${GREEN}Publisher: $pub_id${NC}"
        echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
        echo -e "  Status: ${GREEN}REGISTERED${NC}"
        echo -e "  Allowed Domains: ${BLUE}$domains${NC}"

        # Parse and display domains
        echo ""
        echo "Domain Rules:"
        IFS='|' read -ra DOMAINS <<< "$domains"
        for domain in "${DOMAINS[@]}"; do
            domain=$(echo "$domain" | xargs) # trim whitespace
            if [ "$domain" = "*" ]; then
                echo -e "  • ${YELLOW}$domain${NC} (any domain - permissive!)"
            elif [[ "$domain" == \*.* ]]; then
                echo -e "  • ${BLUE}$domain${NC} (wildcard subdomain)"
            else
                echo -e "  • ${GREEN}$domain${NC} (exact match)"
            fi
        done
    fi
    echo ""
}

# Update publisher domains
update_publisher() {
    local pub_id=$1
    local domains=$2

    if [ -z "$pub_id" ] || [ -z "$domains" ]; then
        echo -e "${RED}Error: Missing arguments${NC}"
        echo ""
        echo "Usage: $0 update <publisher_id> <new_domains>"
        echo ""
        echo "Example:"
        echo "  $0 update pub123 'newdomain.com|*.newdomain.com'"
        exit 1
    fi

    # Check if exists
    local existing=$(docker exec $REDIS_CONTAINER redis-cli HGET $REDIS_KEY "$pub_id" 2>/dev/null)
    if [ -z "$existing" ]; then
        echo -e "${YELLOW}Warning: Publisher '$pub_id' doesn't exist${NC}"
        echo -e "${YELLOW}Use 'add' command to create new publisher${NC}"
        exit 1
    fi

    docker exec $REDIS_CONTAINER redis-cli HSET $REDIS_KEY "$pub_id" "$domains" > /dev/null
    echo -e "${GREEN}✓ Successfully updated publisher${NC}"
    echo ""
    echo -e "  Publisher ID: ${BLUE}$pub_id${NC}"
    echo -e "  Old Domains: ${YELLOW}$existing${NC}"
    echo -e "  New Domains: ${GREEN}$domains${NC}"
}

# Export publishers to JSON
export_publishers() {
    echo -e "${BLUE}Exporting publishers...${NC}"

    local output=$(docker exec $REDIS_CONTAINER redis-cli HGETALL $REDIS_KEY 2>/dev/null)

    if [ -z "$output" ]; then
        echo -e "${YELLOW}No publishers to export${NC}"
        return
    fi

    local filename="publishers-export-$(date +%Y%m%d-%H%M%S).json"

    echo "{" > "$filename"
    echo "  \"exported_at\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"," >> "$filename"
    echo "  \"publishers\": {" >> "$filename"

    local first=true
    echo "$output" | awk '
        NR%2==1 { pub=$0 }
        NR%2==0 {
            if (!first) printf ",\n"
            printf "    \"%s\": \"%s\"", pub, $0
            first=0
        }' >> "$filename"

    echo "" >> "$filename"
    echo "  }" >> "$filename"
    echo "}" >> "$filename"

    echo -e "${GREEN}✓ Exported to: $filename${NC}"
}

# Import publishers from JSON
import_publishers() {
    local filename=$1

    if [ -z "$filename" ]; then
        echo -e "${RED}Error: Missing filename${NC}"
        echo ""
        echo "Usage: $0 import <filename.json>"
        exit 1
    fi

    if [ ! -f "$filename" ]; then
        echo -e "${RED}Error: File not found: $filename${NC}"
        exit 1
    fi

    echo -e "${YELLOW}Importing publishers from: $filename${NC}"
    echo ""

    # Simple JSON parsing (requires jq if available, otherwise manual)
    if command -v jq &> /dev/null; then
        local count=0
        while IFS="=" read -r pub_id domains; do
            if [ -n "$pub_id" ] && [ -n "$domains" ]; then
                docker exec $REDIS_CONTAINER redis-cli HSET $REDIS_KEY "$pub_id" "$domains" > /dev/null
                echo -e "  ${GREEN}✓${NC} Imported: $pub_id"
                count=$((count + 1))
            fi
        done < <(jq -r '.publishers | to_entries[] | "\(.key)=\(.value)"' "$filename")

        echo ""
        echo -e "${GREEN}✓ Imported $count publisher(s)${NC}"
    else
        echo -e "${RED}Error: jq not installed${NC}"
        echo "Install jq: apt install jq"
    fi
}

# Show help
show_help() {
    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}Catalyst Publisher Management${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
    echo ""
    echo "Usage: $0 <command> [options]"
    echo ""
    echo -e "${GREEN}Commands:${NC}"
    echo "  list, ls                    List all registered publishers"
    echo "  add <id> <domains>          Add new publisher"
    echo "  remove, rm <id>             Remove publisher"
    echo "  check <id>                  Check specific publisher"
    echo "  update <id> <domains>       Update publisher domains"
    echo "  export                      Export publishers to JSON"
    echo "  import <file.json>          Import publishers from JSON"
    echo ""
    echo -e "${GREEN}Domain Format:${NC}"
    echo "  Single domain:              example.com"
    echo "  Multiple domains:           example.com|cdn.example.com"
    echo "  Wildcard subdomain:         *.example.com"
    echo "  Allow any domain:           * (testing only!)"
    echo ""
    echo -e "${GREEN}Examples:${NC}"
    echo "  $0 list"
    echo "  $0 add pub123 'example.com'"
    echo "  $0 add pub456 'example.com|*.example.com'"
    echo "  $0 check pub123"
    echo "  $0 update pub123 'newdomain.com|*.newdomain.com'"
    echo "  $0 remove pub123"
    echo "  $0 export"
    echo "  $0 import publishers.json"
    echo ""
    echo -e "${YELLOW}Note: Changes take effect immediately (no restart needed)${NC}"
    echo ""
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
    export)
        export_publishers
        ;;
    import)
        import_publishers "$2"
        ;;
    help|-h|--help)
        show_help
        ;;
    *)
        show_help
        ;;
esac
