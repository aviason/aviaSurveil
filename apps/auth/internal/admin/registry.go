package admin

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"net"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidClient  = errors.New("provider client configuration is invalid")
	ErrClientNotFound = errors.New("provider client not found")
	ErrClientInactive = errors.New("provider client is inactive")
	ErrInvalidKey     = errors.New("provider signing key is invalid")
	ErrKeyNotFound    = errors.New("provider signing key not found")
)

var clientIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

type Client struct {
	ID                     string
	RedirectURIs           []string
	PostLogoutRedirectURIs []string
	Scopes                 []string
	SecretHash             [32]byte
	Active                 bool
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type ClientRegistry struct {
	mu      sync.RWMutex
	clients map[string]Client
	clock   func() time.Time
}

func NewClientRegistry(clock func() time.Time) *ClientRegistry {
	if clock == nil {
		clock = time.Now
	}
	return &ClientRegistry{clients: make(map[string]Client), clock: clock}
}

func (registry *ClientRegistry) Register(id, secret string, redirects, postLogout []string, scopes []string) (Client, error) {
	if !clientIDPattern.MatchString(id) || strings.TrimSpace(secret) == "" || len(redirects) == 0 {
		return Client{}, ErrInvalidClient
	}
	if err := validateRedirects(redirects, false); err != nil {
		return Client{}, err
	}
	if err := validateRedirects(postLogout, true); err != nil {
		return Client{}, err
	}
	now := registry.clock().UTC()
	client := Client{ID: id, RedirectURIs: append([]string(nil), redirects...), PostLogoutRedirectURIs: append([]string(nil), postLogout...), Scopes: append([]string(nil), scopes...), SecretHash: sha256.Sum256([]byte(secret)), Active: true, CreatedAt: now, UpdatedAt: now}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.clients[id]; exists {
		return Client{}, ErrInvalidClient
	}
	registry.clients[id] = cloneClient(client)
	return cloneClient(client), nil
}

func (registry *ClientRegistry) Get(id string) (Client, error) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	client, ok := registry.clients[id]
	if !ok {
		return Client{}, ErrClientNotFound
	}
	return cloneClient(client), nil
}

func (registry *ClientRegistry) Authenticate(id, secret string) error {
	client, err := registry.Get(id)
	if err != nil {
		return err
	}
	if !client.Active {
		return ErrClientInactive
	}
	hash := sha256.Sum256([]byte(secret))
	if !hmac.Equal(client.SecretHash[:], hash[:]) {
		return ErrClientNotFound
	}
	return nil
}

func (registry *ClientRegistry) Revoke(id string) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	client, ok := registry.clients[id]
	if !ok {
		return ErrClientNotFound
	}
	client.Active = false
	client.UpdatedAt = registry.clock().UTC()
	registry.clients[id] = client
	return nil
}

func validateRedirects(values []string, optional bool) error {
	if len(values) == 0 && optional {
		return nil
	}
	for _, raw := range values {
		parsed, err := url.Parse(raw)
		if err != nil || !parsed.IsAbs() || parsed.User != nil || parsed.Fragment != "" || strings.Contains(raw, "*") || parsed.Host == "" {
			return ErrInvalidClient
		}
		if parsed.Scheme != "https" && !isLoopbackHTTP(parsed) {
			return ErrInvalidClient
		}
	}
	return nil
}

func isLoopbackHTTP(parsed *url.URL) bool {
	if parsed.Scheme != "http" {
		return false
	}
	host := parsed.Hostname()
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func cloneClient(client Client) Client {
	client.RedirectURIs = append([]string(nil), client.RedirectURIs...)
	client.PostLogoutRedirectURIs = append([]string(nil), client.PostLogoutRedirectURIs...)
	client.Scopes = append([]string(nil), client.Scopes...)
	return client
}

type KeyState string

const (
	KeyActive  KeyState = "active"
	KeyOverlap KeyState = "overlap"
	KeyRetired KeyState = "retired"
)

type KeyRecord struct {
	ID        string
	Algorithm string
	Private   *rsa.PrivateKey
	State     KeyState
	RetireAt  time.Time
	CreatedAt time.Time
}

type KeyRing struct {
	mu    sync.RWMutex
	clock func() time.Time
	keys  map[string]KeyRecord
}

func NewKeyRing(initialID string, initial *rsa.PrivateKey, clock func() time.Time) (*KeyRing, error) {
	if !validKeyID(initialID) || initial == nil || initial.N == nil || initial.N.BitLen() < 2048 {
		return nil, ErrInvalidKey
	}
	if clock == nil {
		clock = time.Now
	}
	now := clock().UTC()
	return &KeyRing{clock: clock, keys: map[string]KeyRecord{initialID: {ID: initialID, Algorithm: "RS256", Private: initial, State: KeyActive, CreatedAt: now}}}, nil
}

func (ring *KeyRing) Rotate(id string, bits int, overlap time.Duration) (KeyRecord, error) {
	if !validKeyID(id) || bits < 2048 || bits > 8192 || overlap <= 0 || overlap > 30*24*time.Hour {
		return KeyRecord{}, ErrInvalidKey
	}
	private, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return KeyRecord{}, err
	}
	now := ring.clock().UTC()
	ring.mu.Lock()
	defer ring.mu.Unlock()
	if _, exists := ring.keys[id]; exists {
		return KeyRecord{}, ErrInvalidKey
	}
	for keyID, key := range ring.keys {
		if key.State == KeyActive {
			key.State = KeyOverlap
			key.RetireAt = now.Add(overlap)
			ring.keys[keyID] = key
		}
	}
	record := KeyRecord{ID: id, Algorithm: "RS256", Private: private, State: KeyActive, CreatedAt: now}
	ring.keys[id] = record
	return cloneKey(record), nil
}

func (ring *KeyRing) Retire(id string) error {
	ring.mu.Lock()
	defer ring.mu.Unlock()
	key, ok := ring.keys[id]
	if !ok {
		return ErrKeyNotFound
	}
	if key.State == KeyActive {
		return ErrInvalidKey
	}
	key.State = KeyRetired
	ring.keys[id] = key
	return nil
}

func (ring *KeyRing) Active() (KeyRecord, error) {
	ring.mu.RLock()
	defer ring.mu.RUnlock()
	for _, key := range ring.keys {
		if key.State == KeyActive {
			return cloneKey(key), nil
		}
	}
	return KeyRecord{}, ErrKeyNotFound
}

func (ring *KeyRing) VerificationKeys() []KeyRecord {
	now := ring.clock().UTC()
	ring.mu.RLock()
	defer ring.mu.RUnlock()
	result := make([]KeyRecord, 0, len(ring.keys))
	for _, key := range ring.keys {
		if key.State == KeyActive || (key.State == KeyOverlap && now.Before(key.RetireAt)) {
			result = append(result, cloneKey(key))
		}
	}
	return result
}

func (ring *KeyRing) Ready() error {
	key, err := ring.Active()
	if err != nil || key.Algorithm != "RS256" || key.Private == nil || key.Private.N.BitLen() < 2048 {
		return ErrInvalidKey
	}
	if len(ring.VerificationKeys()) == 0 {
		return ErrInvalidKey
	}
	return nil
}

func validKeyID(id string) bool {
	return clientIDPattern.MatchString(id)
}

func cloneKey(key KeyRecord) KeyRecord { return key }
