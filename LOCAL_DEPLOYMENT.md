# 🚀 Local Deployment - TNE Catalyst Server

## ✅ Deployment Status: RUNNING

**Server URL:** http://localhost:8000  
**Process ID:** `cat server.pid`  
**Log File:** server.log

---

## 📊 Server Health Check

```bash
✅ Health:  http://localhost:8000/health
✅ Status:  http://localhost:8000/status
✅ Bidders: http://localhost:8000/info/bidders
```

**Available Bidders:**
- AppNexus
- Demo
- Rubicon
- PubMatic

---

## 🔧 Configuration

**Current Setup:**
- Port: 8000
- IDR: Disabled (optional ML component)
- Redis: Not connected (optional caching)
- PostgreSQL: Not connected (optional storage)
- CORS: Permissive (all origins allowed)
- Log Level: Info
- Debug Endpoints: Enabled

The server runs in **standalone mode** without Redis/PostgreSQL. This is fine for:
- Local development
- Testing
- API exploration

---

## 📡 Available Endpoints

### Health & Status
- `GET /health` - Health check
- `GET /health/ready` - Readiness check
- `GET /status` - Server status

### Auction
- `POST /openrtb2/auction` - OpenRTB 2.x auction endpoint
- `POST /openrtb2/video/auction` - Video auction endpoint

### Cookie Sync
- `GET /cookie_sync` - Cookie sync status
- `POST /setuid` - Set user ID

### Information
- `GET /info/bidders` - List available bidders
- `GET /admin/dashboard` - Admin dashboard
- `GET /admin/circuit-breakers` - Circuit breaker status

### Monitoring
- `GET /metrics` - Prometheus metrics

---

## 🧪 Test the Server

### 1. Simple Health Check
```bash
curl http://localhost:8000/health
```

### 2. Get Available Bidders
```bash
curl http://localhost:8000/info/bidders
```

### 3. Sample Video Auction Request
```bash
curl -X POST http://localhost:8000/openrtb2/video/auction \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-auction-001",
    "imp": [{
      "id": "1",
      "video": {
        "mimes": ["video/mp4"],
        "minduration": 5,
        "maxduration": 30,
        "protocols": [2, 3, 5, 6],
        "w": 640,
        "h": 480
      }
    }],
    "site": {
      "page": "http://example.com/test"
    },
    "device": {
      "ua": "Mozilla/5.0",
      "ip": "192.168.1.1"
    }
  }'
```

### 4. View Metrics
```bash
curl http://localhost:8000/metrics | grep catalyst
```

---

## 🔍 Monitoring

### View Live Logs
```bash
tail -f server.log
```

### Check Process
```bash
ps aux | grep catalyst-server
```

### Check Port
```bash
lsof -i :8000
```

---

## 🛑 Stop the Server

```bash
# Graceful shutdown
kill $(cat server.pid)

# Or force kill
kill -9 $(cat server.pid)

# Remove PID file
rm server.pid
```

---

## 🐳 Optional: Add Redis & PostgreSQL

If you want full functionality, start Docker Desktop and run:

```bash
# Start Redis
docker run -d --name catalyst-redis -p 6379:6379 redis:7-alpine

# Start PostgreSQL
docker run -d --name catalyst-postgres \
  -e POSTGRES_DB=catalyst_dev \
  -e POSTGRES_USER=catalyst \
  -e POSTGRES_PASSWORD=dev123 \
  -p 5432:5432 \
  postgres:16-alpine

# Restart the server to pick up the databases
kill $(cat server.pid)
./catalyst-server > server.log 2>&1 &
echo $! > server.pid
```

---

## 📝 Configuration File

Location: `.env.local`

Edit this file to change server settings, then restart the server.

---

## 🎯 Next Steps

1. ✅ Server is running and healthy
2. Test auction endpoints with sample requests
3. View Prometheus metrics at /metrics
4. Explore admin dashboard at /admin/dashboard
5. Add Redis/PostgreSQL for full features (optional)

---

**Deployment Time:** $(date)  
**Status:** ✅ OPERATIONAL
