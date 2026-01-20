#!/bin/bash
# Generate Production Secrets
# Creates strong, random secrets for all production credentials

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "========================================="
echo "Production Secrets Generator"
echo "========================================="
echo ""

# Check if openssl is available
if ! command -v openssl &> /dev/null; then
    echo -e "${RED}❌ openssl not found${NC}"
    echo "Install with: apt-get install openssl  (or)  brew install openssl"
    exit 1
fi

# Function to generate random password
generate_password() {
    local length=${1:-32}
    openssl rand -base64 $length | tr -d "=+/" | cut -c1-$length
}

# Function to generate hex secret
generate_hex() {
    local length=${1:-64}
    openssl rand -hex $((length / 2))
}

echo -e "${BLUE}Generating strong secrets...${NC}"
echo ""

# Generate secrets
DB_PASSWORD=$(generate_password 32)
REDIS_PASSWORD=$(generate_password 32)
JWT_SECRET=$(generate_password 64)
API_SECRET=$(generate_hex 64)
SESSION_SECRET=$(generate_hex 32)
ENCRYPTION_KEY=$(generate_hex 32)

echo "========================================="
echo "Generated Secrets"
echo "========================================="
echo ""
echo -e "${GREEN}✅ Database Password:${NC}"
echo "DB_PASSWORD=${DB_PASSWORD}"
echo ""
echo -e "${GREEN}✅ Redis Password:${NC}"
echo "REDIS_PASSWORD=${REDIS_PASSWORD}"
echo ""
echo -e "${GREEN}✅ JWT Secret:${NC}"
echo "JWT_SECRET=${JWT_SECRET}"
echo ""
echo -e "${GREEN}✅ API Secret:${NC}"
echo "API_SECRET=${API_SECRET}"
echo ""
echo -e "${GREEN}✅ Session Secret:${NC}"
echo "SESSION_SECRET=${SESSION_SECRET}"
echo ""
echo -e "${GREEN}✅ Encryption Key:${NC}"
echo "ENCRYPTION_KEY=${ENCRYPTION_KEY}"
echo ""

# Ask if user wants to update .env.production
echo "========================================="
echo -e "${YELLOW}Update .env.production automatically?${NC}"
echo "This will replace CHANGE_ME values in .env.production"
echo ""
read -p "Update .env.production? (y/N): " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    ENV_FILE=".env.production"

    if [ ! -f "$ENV_FILE" ]; then
        echo -e "${RED}❌ .env.production not found${NC}"
        exit 1
    fi

    # Backup original
    cp "$ENV_FILE" "${ENV_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
    echo -e "${GREEN}✅ Backed up to ${ENV_FILE}.backup.$(date +%Y%m%d_%H%M%S)${NC}"

    # Update secrets
    sed -i.tmp "s/^DB_PASSWORD=.*/DB_PASSWORD=${DB_PASSWORD}/" "$ENV_FILE"
    sed -i.tmp "s/^REDIS_PASSWORD=.*/REDIS_PASSWORD=${REDIS_PASSWORD}/" "$ENV_FILE"

    # Add new secrets if they don't exist
    if ! grep -q "^JWT_SECRET=" "$ENV_FILE"; then
        echo "" >> "$ENV_FILE"
        echo "# Authentication Secrets" >> "$ENV_FILE"
        echo "JWT_SECRET=${JWT_SECRET}" >> "$ENV_FILE"
    else
        sed -i.tmp "s/^JWT_SECRET=.*/JWT_SECRET=${JWT_SECRET}/" "$ENV_FILE"
    fi

    if ! grep -q "^API_SECRET=" "$ENV_FILE"; then
        echo "API_SECRET=${API_SECRET}" >> "$ENV_FILE"
    else
        sed -i.tmp "s/^API_SECRET=.*/API_SECRET=${API_SECRET}/" "$ENV_FILE"
    fi

    if ! grep -q "^SESSION_SECRET=" "$ENV_FILE"; then
        echo "SESSION_SECRET=${SESSION_SECRET}" >> "$ENV_FILE"
    else
        sed -i.tmp "s/^SESSION_SECRET=.*/SESSION_SECRET=${SESSION_SECRET}/" "$ENV_FILE"
    fi

    if ! grep -q "^ENCRYPTION_KEY=" "$ENV_FILE"; then
        echo "ENCRYPTION_KEY=${ENCRYPTION_KEY}" >> "$ENV_FILE"
    else
        sed -i.tmp "s/^ENCRYPTION_KEY=.*/ENCRYPTION_KEY=${ENCRYPTION_KEY}/" "$ENV_FILE"
    fi

    # Clean up temp files
    rm -f "${ENV_FILE}.tmp"

    echo -e "${GREEN}✅ Updated .env.production${NC}"

    # Verify no CHANGE_ME left
    CHANGE_ME_COUNT=$(grep -c "CHANGE_ME" "$ENV_FILE" || true)
    if [ $CHANGE_ME_COUNT -gt 0 ]; then
        echo -e "${YELLOW}⚠️  Warning: ${CHANGE_ME_COUNT} CHANGE_ME values still remain${NC}"
        grep -n "CHANGE_ME" "$ENV_FILE"
    else
        echo -e "${GREEN}✅ No CHANGE_ME values remaining${NC}"
    fi
else
    echo ""
    echo "Secrets not saved. Copy the values above to .env.production manually."
fi

# Generate secure config snippet
echo ""
echo "========================================="
echo "Secure Storage (for secrets manager)"
echo "========================================="
echo ""
echo "Save these to your secrets manager (AWS Secrets Manager, Vault, etc.):"
echo ""

cat <<EOF
{
  "catalyst-production": {
    "DB_PASSWORD": "${DB_PASSWORD}",
    "REDIS_PASSWORD": "${REDIS_PASSWORD}",
    "JWT_SECRET": "${JWT_SECRET}",
    "API_SECRET": "${API_SECRET}",
    "SESSION_SECRET": "${SESSION_SECRET}",
    "ENCRYPTION_KEY": "${ENCRYPTION_KEY}"
  }
}
EOF

echo ""
echo "========================================="
echo "Security Recommendations"
echo "========================================="
echo ""
echo "✅ Store secrets in a secrets manager (AWS Secrets Manager, HashiCorp Vault)"
echo "✅ Use environment variable injection at runtime"
echo "✅ Rotate secrets every 90 days"
echo "✅ Never commit secrets to git"
echo "✅ Use different secrets for dev/staging/production"
echo "✅ Enable audit logging for secret access"
echo ""

# Generate password strength report
echo "========================================="
echo "Password Strength Report"
echo "========================================="
echo ""
echo "All secrets generated with cryptographically secure random bytes:"
echo "  • DB_PASSWORD: 32 characters (base64)"
echo "  • REDIS_PASSWORD: 32 characters (base64)"
echo "  • JWT_SECRET: 64 characters (base64)"
echo "  • API_SECRET: 64 characters (hex)"
echo "  • SESSION_SECRET: 32 characters (hex)"
echo "  • ENCRYPTION_KEY: 32 characters (hex)"
echo ""
echo "Estimated entropy:"
echo "  • Base64 (32 chars): ~192 bits"
echo "  • Base64 (64 chars): ~384 bits"
echo "  • Hex (32 chars): ~128 bits"
echo "  • Hex (64 chars): ~256 bits"
echo ""
echo -e "${GREEN}All secrets exceed minimum security requirements.${NC}"
echo ""

echo "========================================="
echo "Done!"
echo "========================================="
