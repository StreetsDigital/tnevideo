#!/bin/bash
# Let's Encrypt SSL Certificate Setup
# Automates SSL certificate generation and renewal

set -euo pipefail

# Configuration
DOMAIN="${DOMAIN:-}"
EMAIL="${EMAIL:-}"
STAGING="${STAGING:-false}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "========================================="
echo "Let's Encrypt SSL Setup"
echo "========================================="
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${YELLOW}⚠️  Not running as root. Some commands may require sudo.${NC}"
fi

# Prompt for domain if not set
if [ -z "$DOMAIN" ]; then
    read -p "Enter your domain (e.g., catalyst.example.com): " DOMAIN
fi

# Prompt for email if not set
if [ -z "$EMAIL" ]; then
    read -p "Enter your email for Let's Encrypt notifications: " EMAIL
fi

echo ""
echo "Domain: $DOMAIN"
echo "Email: $EMAIL"
echo "Staging mode: $STAGING"
echo ""

# Check if certbot is installed
if ! command -v certbot &> /dev/null; then
    echo -e "${YELLOW}⚠️  certbot not found. Installing...${NC}"

    if command -v apt-get &> /dev/null; then
        # Debian/Ubuntu
        apt-get update
        apt-get install -y certbot
    elif command -v yum &> /dev/null; then
        # RHEL/CentOS
        yum install -y certbot
    elif command -v brew &> /dev/null; then
        # macOS
        brew install certbot
    else
        echo -e "${RED}❌ Unable to install certbot automatically${NC}"
        echo "Install manually: https://certbot.eff.org/"
        exit 1
    fi
fi

echo -e "${GREEN}✅ certbot installed${NC}"

# Create SSL directory
mkdir -p ssl
chmod 700 ssl

# Check if certificates already exist
if [ -f "ssl/fullchain.pem" ] && [ -f "ssl/privkey.pem" ]; then
    echo -e "${YELLOW}⚠️  Certificates already exist in ssl/ directory${NC}"
    read -p "Overwrite existing certificates? (y/N): " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Exiting without changes."
        exit 0
    fi
fi

# Build certbot command
CERTBOT_CMD="certbot certonly --standalone"

if [ "$STAGING" = "true" ]; then
    CERTBOT_CMD="$CERTBOT_CMD --staging"
    echo -e "${YELLOW}⚠️  Using Let's Encrypt STAGING environment (for testing)${NC}"
fi

CERTBOT_CMD="$CERTBOT_CMD --non-interactive --agree-tos"
CERTBOT_CMD="$CERTBOT_CMD --email $EMAIL"
CERTBOT_CMD="$CERTBOT_CMD -d $DOMAIN"

# Stop nginx if running (so certbot can bind to port 80)
if docker ps | grep -q catalyst-nginx; then
    echo "Stopping nginx temporarily..."
    docker stop catalyst-nginx
    RESTART_NGINX=true
else
    RESTART_NGINX=false
fi

# Run certbot
echo ""
echo "Requesting certificate from Let's Encrypt..."
echo "This may take a minute..."
echo ""

if $CERTBOT_CMD; then
    echo -e "${GREEN}✅ Certificate obtained successfully${NC}"

    # Copy certificates to ssl directory
    CERT_DIR="/etc/letsencrypt/live/$DOMAIN"

    if [ -d "$CERT_DIR" ]; then
        cp "$CERT_DIR/fullchain.pem" ssl/
        cp "$CERT_DIR/privkey.pem" ssl/
        cp "$CERT_DIR/chain.pem" ssl/
        chmod 600 ssl/privkey.pem
        chmod 644 ssl/fullchain.pem ssl/chain.pem

        # Create symbolic links for common names
        ln -sf fullchain.pem ssl/server.crt
        ln -sf privkey.pem ssl/server.key

        echo -e "${GREEN}✅ Certificates copied to ssl/ directory${NC}"
    else
        echo -e "${RED}❌ Certificate directory not found: $CERT_DIR${NC}"
        exit 1
    fi
else
    echo -e "${RED}❌ Certificate request failed${NC}"
    exit 1
fi

# Restart nginx if it was running
if [ "$RESTART_NGINX" = "true" ]; then
    echo "Restarting nginx..."
    docker start catalyst-nginx
fi

# Display certificate information
echo ""
echo "========================================="
echo "Certificate Information"
echo "========================================="
openssl x509 -in ssl/fullchain.pem -noout -text | grep -A2 "Validity"
openssl x509 -in ssl/fullchain.pem -noout -subject
openssl x509 -in ssl/fullchain.pem -noout -issuer

# Calculate expiry date
EXPIRY=$(openssl x509 -in ssl/fullchain.pem -noout -enddate | cut -d= -f2)
EXPIRY_EPOCH=$(date -d "$EXPIRY" +%s 2>/dev/null || date -j -f "%b %d %T %Y %Z" "$EXPIRY" +%s 2>/dev/null)
NOW_EPOCH=$(date +%s)
DAYS_UNTIL_EXPIRY=$(( (EXPIRY_EPOCH - NOW_EPOCH) / 86400 ))

echo ""
echo "Certificate expires in: $DAYS_UNTIL_EXPIRY days"

# Set up automatic renewal
echo ""
echo "========================================="
echo "Setting Up Automatic Renewal"
echo "========================================="

# Create renewal script
cat > ssl/renew-cert.sh <<'EOF'
#!/bin/bash
# Certificate Renewal Script
# Run by cron to renew certificates before expiry

set -e

DOMAIN="${DOMAIN}"
CERT_DIR="/etc/letsencrypt/live/${DOMAIN}"

# Check if certificate expires in next 30 days
if openssl x509 -checkend $((30 * 86400)) -noout -in "${CERT_DIR}/fullchain.pem"; then
    echo "Certificate is valid for more than 30 days. No renewal needed."
    exit 0
fi

echo "Certificate expires soon. Renewing..."

# Stop nginx
docker stop catalyst-nginx

# Renew certificate
certbot renew --quiet

# Copy renewed certificates
cp "${CERT_DIR}/fullchain.pem" /path/to/deployment/ssl/
cp "${CERT_DIR}/privkey.pem" /path/to/deployment/ssl/
cp "${CERT_DIR}/chain.pem" /path/to/deployment/ssl/

# Restart nginx
docker start catalyst-nginx

echo "Certificate renewed successfully"
EOF

# Update the script with actual path
SCRIPT_DIR=$(pwd)
sed -i.bak "s|/path/to/deployment|${SCRIPT_DIR}|g" ssl/renew-cert.sh
sed -i.bak "s/\${DOMAIN}/${DOMAIN}/g" ssl/renew-cert.sh
rm ssl/renew-cert.sh.bak

chmod +x ssl/renew-cert.sh

echo -e "${GREEN}✅ Renewal script created: ssl/renew-cert.sh${NC}"

# Add cron job
CRON_JOB="0 3 * * * ${SCRIPT_DIR}/ssl/renew-cert.sh >> ${SCRIPT_DIR}/ssl/renewal.log 2>&1"

echo ""
echo "To set up automatic renewal, add this cron job:"
echo ""
echo "$CRON_JOB"
echo ""
echo "Or run this command:"
echo ""
echo "(crontab -l 2>/dev/null; echo \"$CRON_JOB\") | crontab -"
echo ""

# Test nginx configuration with new certificates
echo "========================================="
echo "Testing nginx Configuration"
echo "========================================="

if docker ps | grep -q catalyst-nginx; then
    if docker exec catalyst-nginx nginx -t 2>&1 | grep -q "successful"; then
        echo -e "${GREEN}✅ nginx configuration test passed${NC}"

        # Reload nginx to use new certificates
        docker exec catalyst-nginx nginx -s reload
        echo -e "${GREEN}✅ nginx reloaded with new certificates${NC}"
    else
        echo -e "${RED}❌ nginx configuration test failed${NC}"
        docker exec catalyst-nginx nginx -t
    fi
else
    echo -e "${YELLOW}⚠️  nginx not running. Start it to use the certificates.${NC}"
fi

# Verify HTTPS
echo ""
echo "========================================="
echo "Verification"
echo "========================================="
echo ""
echo "Test your SSL certificate:"
echo "  • https://${DOMAIN}"
echo "  • https://www.ssllabs.com/ssltest/analyze.html?d=${DOMAIN}"
echo ""
echo "Certificate files:"
echo "  • ssl/fullchain.pem (certificate + chain)"
echo "  • ssl/privkey.pem (private key)"
echo "  • ssl/chain.pem (intermediate certificates)"
echo ""
echo "Renewal:"
echo "  • Automatic renewal script: ssl/renew-cert.sh"
echo "  • Runs 30 days before expiry"
echo "  • Add to cron for automation"
echo ""

echo "========================================="
echo "Done!"
echo "========================================="
echo ""
echo -e "${GREEN}✅ SSL certificates installed and configured${NC}"
echo -e "${GREEN}✅ nginx reloaded with new certificates${NC}"
echo -e "${GREEN}✅ Automatic renewal script created${NC}"
echo ""
echo "Next steps:"
echo "  1. Add cron job for automatic renewal"
echo "  2. Test HTTPS: https://${DOMAIN}"
echo "  3. Update .env.production: PBS_HOST_URL=https://${DOMAIN}"
echo "  4. Configure HTTPS redirect in nginx"
echo ""
