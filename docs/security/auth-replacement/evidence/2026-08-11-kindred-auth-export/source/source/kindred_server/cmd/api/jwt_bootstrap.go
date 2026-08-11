package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"kindred_server/pkg/config"
	"kindred_server/pkg/security"
)

// loadJWTPrivateKey resolves the JWT signing key from the configured
// sources, falling back to a generated dev key file when nothing is set.
func loadJWTPrivateKey(ctx context.Context, cfg config.Config) (*rsa.PrivateKey, error) {
	if cfg.JWTPrivateKeyPEM != "" {
		return security.LoadRSAPrivateKeyPEM([]byte(cfg.JWTPrivateKeyPEM))
	}
	if cfg.JWTPrivateKeySSMPath != "" {
		pem, err := fetchSSMSecureString(ctx, cfg.AWSRegion, cfg.JWTPrivateKeySSMPath)
		if err != nil {
			return nil, fmt.Errorf("fetch JWT key from SSM: %w", err)
		}
		return security.LoadRSAPrivateKeyPEM([]byte(pem))
	}
	path := cfg.JWTPrivateKeyPath
	if path == "" {
		// Default dev location — generated on first run, gitignored.
		path = "infrastructure/docker/data/dev_jwt_private.pem"
	}
	if pem, err := os.ReadFile(path); err == nil {
		return security.LoadRSAPrivateKeyPEM(pem)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read JWT key %s: %w", path, err)
	}
	if cfg.Environment != "dev" {
		return nil, fmt.Errorf("no JWT private key configured (env=%s)", cfg.Environment)
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate dev JWT key: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("mkdir JWT key dir: %w", err)
	}
	if err := os.WriteFile(path, security.EncodeRSAPrivateKeyPEM(priv), 0o600); err != nil {
		return nil, fmt.Errorf("write JWT key: %w", err)
	}
	return priv, nil
}

func fetchSSMSecureString(ctx context.Context, region, name string) (string, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return "", err
	}
	client := ssm.NewFromConfig(awsCfg)
	out, err := client.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           &name,
		WithDecryption: ptr(true),
	})
	if err != nil {
		return "", err
	}
	if out.Parameter == nil || out.Parameter.Value == nil {
		return "", errors.New("ssm parameter has no value")
	}
	return *out.Parameter.Value, nil
}

func ptr[T any](v T) *T { return &v }
