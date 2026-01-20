## Performance Tuning Guide - TNE Catalyst

### Quick Wins

#### 1. Database Connection Pooling
```bash
# In .env.production
DB_MAX_OPEN_CONNS=100      # Max connections (default: 100)
DB_MAX_IDLE_CONNS=25       # Idle connections (default: 25)
DB_CONN_MAX_LIFETIME=600s  # Connection lifetime (default: 10m)
```

**Tuning:**
- High traffic: Increase MAX_OPEN_CONNS to 200
- Low latency: Increase MAX_IDLE_CONNS to 50
- Monitor: `catalyst_database_connections_*` metrics

#### 2. Redis Configuration
```bash
# In docker-compose.yml
redis:
  command: >
    redis-server
    --maxmemory 2gb              # Increase from 1gb for high traffic
    --maxmemory-policy allkeys-lru
    --tcp-backlog 511            # Connection queue size
    --timeout 0                  # No idle timeout
    --tcp-keepalive 300         # Keep connections alive
```

#### 3. Go Runtime Tuning
```bash
# In .env.production
GOMAXPROCS=0       # Use all CPUs (0 = auto)
GOGC=100           # GC aggressiveness (lower = more frequent, less memory)
GOMEMLIMIT=3584MiB # 3.5GB limit (leave 512MB for OS)
```

**Tuning for high throughput:**
```bash
GOGC=200           # Less frequent GC, higher throughput
GOMEMLIMIT=7168MiB # 7GB if container has 8GB
```

#### 4. Rate Limiting
```bash
# Adjust based on capacity
RATE_LIMIT_REQUESTS=10000  # Requests per minute per IP
RATE_LIMIT_BURST=100       # Burst allowance
```

### Response Time Optimization

#### Target Latencies
| Endpoint | P50 | P95 | P99 |
|----------|-----|-----|-----|
| /health | <10ms | <20ms | <50ms |
| /health/ready | <30ms | <50ms | <100ms |
| /openrtb2/auction | <100ms | <200ms | <300ms |

#### Auction Optimization

**1. Parallel Bidder Requests**
```go
// Already implemented with goroutines
// Tune timeout in .env:
BIDDER_TIMEOUT=200ms  # Max wait per bidder
```

**2. Database Query Optimization**
```sql
-- Ensure indexes exist
CREATE INDEX idx_publishers_api_key ON publishers(api_key);
CREATE INDEX idx_bidders_bidder_id ON bidders(bidder_id);
CREATE INDEX idx_bidders_active ON bidders(active) WHERE active = true;
```

**3. Redis Caching**
```bash
# Cache TTLs (in .env)
PUBLISHER_CACHE_TTL=300    # 5 minutes
BIDDER_CACHE_TTL=300       # 5 minutes
CONFIG_CACHE_TTL=60        # 1 minute
```

### Memory Optimization

#### Current Limits
```yaml
# docker-compose.yml
catalyst:
  deploy:
    resources:
      limits:
        memory: 4G    # Maximum
      reservations:
        memory: 1G    # Guaranteed
```

#### Memory Profiling
```bash
# Enable pprof endpoint
curl http://localhost:8000/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Commands in pprof:
# > top10        # Top 10 memory consumers
# > list <func>  # Source code with allocations
# > web          # Visual graph
```

#### Common Memory Issues

**1. Goroutine Leaks**
```bash
# Check goroutine count
curl http://localhost:8000/metrics | grep go_goroutines

# Should be < 10,000 under normal load
# Alert if > 10,000 (see prometheus/alert-rules.yml)
```

**2. Response Body Not Closed**
```go
// Good practice (already implemented)
defer resp.Body.Close()
```

### CPU Optimization

#### Current Limits
```yaml
catalyst:
  deploy:
    resources:
      limits:
        cpus: '2.0'   # 2 CPU cores max
      reservations:
        cpus: '0.5'   # 0.5 cores guaranteed
```

#### CPU Profiling
```bash
# Capture 30-second CPU profile
curl http://localhost:8000/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# Commands:
# > top10        # Hottest functions
# > list <func>  # Source code
# > web          # Flame graph
```

#### Optimization Tips

**1. JSON Marshaling**
```go
// Use jsoniter for 2-3x faster JSON operations (optional)
import jsoniter "github.com/json-iterator/go"
var json = jsoniter.ConfigCompatibleWithStandardLibrary
```

**2. String Concatenation**
```go
// Use strings.Builder for repeated concatenation
var builder strings.Builder
builder.WriteString(part1)
builder.WriteString(part2)
result := builder.String()
```

### Network Optimization

#### Keep-Alive Connections
```yaml
# nginx.conf (already configured)
keepalive_timeout 65;
keepalive_requests 1000;
```

#### Compression
```yaml
# nginx.conf
gzip on;
gzip_vary on;
gzip_min_length 1024;
gzip_types application/json text/plain;
```

### Database Optimization

#### Connection Pool Monitoring
```prometheus
# Check pool utilization
catalyst_database_connections_in_use / catalyst_database_connections_max

# Alert if > 90%
```

#### Query Performance
```sql
-- Enable slow query logging
ALTER SYSTEM SET log_min_duration_statement = 1000; -- Log queries > 1s
SELECT pg_reload_conf();

-- Check slow queries
SELECT query, mean_exec_time, calls
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;
```

#### Vacuum and Analyze
```bash
# Run weekly
docker exec catalyst-postgres psql -U catalyst -d catalyst -c "VACUUM ANALYZE;"

# Add to cron:
# 0 3 * * 0 docker exec catalyst-postgres psql -U catalyst -d catalyst -c "VACUUM ANALYZE;"
```

### Load Testing

#### Apache Bench
```bash
# Test health endpoint
ab -n 10000 -c 100 http://localhost:8000/health

# Test auction endpoint (with valid request)
ab -n 1000 -c 50 -p request.json -T application/json \
   http://localhost:8000/openrtb2/auction
```

#### vegeta (recommended)
```bash
# Install
go install github.com/tsenart/vegeta@latest

# Test at 1000 req/s for 30 seconds
echo "GET http://localhost:8000/health" | \
  vegeta attack -rate 1000 -duration 30s | \
  vegeta report

# POST requests
echo "POST http://localhost:8000/openrtb2/auction
Content-Type: application/json
@request.json" | \
  vegeta attack -rate 100 -duration 30s | \
  vegeta report
```

### Monitoring Performance

#### Key Metrics
```prometheus
# Response time
histogram_quantile(0.95, rate(catalyst_http_request_duration_seconds_bucket[5m]))

# Throughput
rate(catalyst_http_requests_total[5m])

# Error rate
rate(catalyst_http_requests_total{status=~"5.."}[5m]) / rate(catalyst_http_requests_total[5m])

# Database latency
histogram_quantile(0.95, rate(catalyst_database_query_duration_seconds_bucket[5m]))

# Goroutines
go_goroutines

# Memory usage
process_resident_memory_bytes / (4 * 1024 * 1024 * 1024) * 100  # % of 4GB
```

### Capacity Planning

#### Current Capacity
- **Throughput:** ~1,250 req/s
- **Concurrent auctions:** ~500
- **Database:** 100 connections
- **Redis:** 1GB cache

#### Scaling Up (Vertical)
```yaml
# Increase resources in docker-compose.yml
catalyst:
  deploy:
    resources:
      limits:
        cpus: '4.0'    # 2 → 4 cores
        memory: 8G      # 4 → 8 GB

# Also increase:
DB_MAX_OPEN_CONNS=200
REDIS_MAXMEMORY=2gb
```

#### Scaling Out (Horizontal)
```yaml
# docker-compose.yml
catalyst:
  deploy:
    replicas: 3  # Run 3 instances

# Add load balancer (nginx upstream)
upstream catalyst {
  least_conn;
  server catalyst-1:8000;
  server catalyst-2:8000;
  server catalyst-3:8000;
}
```

### Performance Checklist

- [ ] Database indexes on frequently queried columns
- [ ] Connection pooling configured appropriately
- [ ] Redis caching enabled with reasonable TTLs
- [ ] GOGC tuned for workload (100-200)
- [ ] GOMAXPROCS set to 0 (use all CPUs)
- [ ] nginx gzip compression enabled
- [ ] HTTP keep-alive enabled
- [ ] Rate limiting configured
- [ ] Metrics collection enabled
- [ ] Alerts configured for high latency
- [ ] Load testing performed at expected traffic + 50%
- [ ] Memory profiling shows no leaks
- [ ] CPU profiling shows no hot spots
- [ ] Database slow query log reviewed
- [ ] Vacuum/analyze running regularly

### Troubleshooting Slow Performance

#### High Response Time
1. Check Grafana dashboard for latency spikes
2. Identify slow endpoint: `catalyst_http_request_duration_seconds`
3. Check database query times
4. Profile CPU/memory for hot spots
5. Review recent code changes

#### High CPU Usage
1. Check goroutine count (possible leak)
2. CPU profile to find hot functions
3. Review JSON marshaling (use jsoniter if needed)
4. Check for inefficient loops/algorithms

#### High Memory Usage
1. Heap profile to find allocations
2. Check for goroutine leaks
3. Review cache sizes
4. Check for unclosed HTTP responses

#### Database Slow
1. Check `pg_stat_statements` for slow queries
2. Verify indexes exist
3. Run VACUUM ANALYZE
4. Check connection pool saturation
5. Consider read replicas for high read volume

---

**Remember:** Profile first, optimize second. Don't guess at performance bottlenecks.
