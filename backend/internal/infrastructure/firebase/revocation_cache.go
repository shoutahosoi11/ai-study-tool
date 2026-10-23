package firebase

import (
	"context"
	"sync"
	"time"

	"firebase.google.com/go/v4/auth"
)

// revocationCheckTTL bounds how stale a "not revoked" verdict may be. Local
// JWT verification alone would let a revoked session live until cookie/token
// expiry (up to 14 days); checking Firebase on every request costs a remote
// call per request. The cache keeps logout-all/disable effective within this
// window while removing the per-request remote lookup.
const revocationCheckTTL = 5 * time.Minute

type revocationCache struct {
	mu    sync.Mutex
	until map[string]time.Time
	ttl   time.Duration
}

func newRevocationCache(ttl time.Duration) *revocationCache {
	return &revocationCache{until: make(map[string]time.Time), ttl: ttl}
}

func (c *revocationCache) fresh(uid string, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	deadline, ok := c.until[uid]
	if !ok {
		return false
	}
	if now.After(deadline) {
		delete(c.until, uid)
		return false
	}
	return true
}

func (c *revocationCache) store(uid string, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.until[uid] = now.Add(c.ttl)
}

// evict drops the cached verdict so the next request re-checks Firebase.
// Called on revoke/delete: this makes revocation instant on the instance
// that performed it; other instances remain bounded by the TTL.
func (c *revocationCache) evict(uid string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.until, uid)
}

// CachedTokenVerifier verifies mobile ID tokens locally and re-checks
// revocation against Firebase at most once per TTL per user. Revocation
// errors surface as the SDK's IsIDTokenRevoked/IsUserDisabled error types,
// so the middleware's client-error classification keeps working.
type CachedTokenVerifier struct {
	client *auth.Client
	cache  *revocationCache
}

func NewCachedTokenVerifier(client *auth.Client) *CachedTokenVerifier {
	return &CachedTokenVerifier{client: client, cache: newRevocationCache(revocationCheckTTL)}
}

func (v *CachedTokenVerifier) VerifyIDToken(ctx context.Context, idToken string) (*auth.Token, error) {
	token, err := v.client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, err
	}
	if v.cache.fresh(token.UID, time.Now()) {
		return token, nil
	}
	token, err = v.client.VerifyIDTokenAndCheckRevoked(ctx, idToken)
	if err != nil {
		return nil, err
	}
	v.cache.store(token.UID, time.Now())
	return token, nil
}
