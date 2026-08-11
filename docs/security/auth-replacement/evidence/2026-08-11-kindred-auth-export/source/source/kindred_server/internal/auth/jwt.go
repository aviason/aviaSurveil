package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func init() {
	jwt.MarshalSingleStringAsArray = false
}

// jwtClaims is the JWT shape we sign. Standard registered claims plus the
// custom fields the application has always carried.
type jwtClaims struct {
	Email        string `json:"email"`
	TokenVersion int    `json:"tv"`
	DeviceID     string `json:"did,omitempty"`
	jwt.RegisteredClaims
}

// Signer wraps an RSA private key and the iss/aud/kid metadata used when
// minting tokens.
type Signer struct {
	priv     *rsa.PrivateKey
	issuer   string
	audience string
	kid      string
}

func NewSigner(priv *rsa.PrivateKey, issuer, audience, kid string) *Signer {
	return &Signer{priv: priv, issuer: issuer, audience: audience, kid: kid}
}

// Sign issues an RS256 JWT with the supplied claims.
func (s *Signer) Sign(userID, email, deviceID string, tokenVersion int, ttl time.Duration) (string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(ttl)
	claims := jwtClaims{
		Email:        email,
		TokenVersion: tokenVersion,
		DeviceID:     deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{s.audience},
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = s.kid
	signed, err := tok.SignedString(s.priv)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign jwt: %w", err)
	}
	return signed, exp, nil
}

// Verifier parses and validates RS256 JWTs against the configured public key.
// It explicitly whitelists RS256 to defend against algorithm-confusion
// attacks (e.g. an attacker submitting an HS256 token signed with the public
// key as the HMAC secret).
type Verifier struct {
	pub      *rsa.PublicKey
	issuer   string
	audience string
}

func NewVerifier(pub *rsa.PublicKey, issuer, audience string) *Verifier {
	return &Verifier{pub: pub, issuer: issuer, audience: audience}
}

// Parse validates the token signature, alg, exp, nbf, iss, aud and returns
// the application claims.
func (v *Verifier) Parse(token string) (Claims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(v.issuer),
		jwt.WithAudience(v.audience),
		jwt.WithExpirationRequired(),
	)
	parsed, err := parser.ParseWithClaims(token, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return v.pub, nil
	})
	if err != nil {
		return Claims{}, err
	}
	c, ok := parsed.Claims.(*jwtClaims)
	if !ok || !parsed.Valid {
		return Claims{}, errors.New("invalid jwt claims")
	}
	exp, _ := c.GetExpirationTime()
	out := Claims{
		UserID:       c.Subject,
		Email:        c.Email,
		TokenVersion: c.TokenVersion,
		DeviceID:     c.DeviceID,
	}
	if exp != nil {
		out.ExpiresAt = exp.UTC()
	}
	return out, nil
}
