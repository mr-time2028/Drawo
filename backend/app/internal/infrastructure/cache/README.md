# `internal/infrastructure/cache`

**Responsibility:** Provide non-relational database / key-value storage connectivity and caching layer adapters.

**Why this package?**
While Redis is the primary non-relational caching solution in production, decoupling the caching layer avoids vendor lock-in and enables seamless switching between Redis, in-memory storage, or future alternative engines (e.g., Memcached).

**Components:**
- `cache.go` — Factory registry for driver initialization based on `config.CacheConfig`.
- `redis.go` — Redis adapter implementing `ports.CacheRepository`.
- `memory.go` — In-memory adapter implementing `ports.CacheRepository` using thread-safe structures for testing or local usage.
