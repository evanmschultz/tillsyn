package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hylla/tillsyn/internal/adapters/livewait/localipc"
)

// runtimeLiveWaitSecretFilename stores the shared per-runtime secret file name for local live-wait brokers.
const runtimeLiveWaitSecretFilename = "livewait.secret"

// errRuntimeLiveWaitSecretInvalid reports a persisted runtime secret that is empty or malformed.
var errRuntimeLiveWaitSecretInvalid = errors.New("runtime live wait secret is invalid")

// newRuntimeLiveWaitBrokerFunc is the injectable runtime broker constructor used by tests.
var newRuntimeLiveWaitBrokerFunc = newRuntimeLiveWaitBroker

// newRuntimeLiveWaitBroker constructs the local cross-process live-wait broker used by real till runs.
func newRuntimeLiveWaitBroker(db *sql.DB, rootDir string) (*localipc.Broker, error) {
	if db == nil {
		return nil, fmt.Errorf("live wait sqlite db is required")
	}
	secret, err := loadOrCreateRuntimeLiveWaitSecret(rootDir)
	if err != nil {
		return nil, err
	}
	broker, err := localipc.NewBroker(db, localipc.Config{Secret: secret})
	if err != nil {
		return nil, fmt.Errorf("construct runtime live wait broker: %w", err)
	}
	return broker, nil
}

// runtimeLiveWaitSecretPath resolves the shared live-wait secret file under one runtime root.
func runtimeLiveWaitSecretPath(rootDir string) string {
	rootDir = strings.TrimSpace(rootDir)
	return filepath.Join(rootDir, runtimeLiveWaitSecretFilename)
}

// loadOrCreateRuntimeLiveWaitSecret loads the shared runtime secret or creates one atomically.
func loadOrCreateRuntimeLiveWaitSecret(rootDir string) (string, error) {
	secretPath := runtimeLiveWaitSecretPath(rootDir)
	if secretPath == runtimeLiveWaitSecretFilename {
		return "", fmt.Errorf("live wait runtime root is required")
	}
	if secret, err := readRuntimeLiveWaitSecret(secretPath); err == nil {
		return secret, nil
	} else if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, errRuntimeLiveWaitSecretInvalid) {
		return "", err
	} else if errors.Is(err, errRuntimeLiveWaitSecretInvalid) {
		if removeErr := os.Remove(secretPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return "", fmt.Errorf("remove invalid live wait secret %q: %w", secretPath, removeErr)
		}
	}
	if err := os.MkdirAll(filepath.Dir(secretPath), 0o755); err != nil {
		return "", fmt.Errorf("create live wait secret directory: %w", err)
	}

	secret, err := generateRuntimeLiveWaitSecret()
	if err != nil {
		return "", err
	}
	if err := writeRuntimeLiveWaitSecret(secretPath, secret); err != nil {
		if errors.Is(err, os.ErrExist) {
			return readRuntimeLiveWaitSecret(secretPath)
		}
		return "", err
	}
	return secret, nil
}

// readRuntimeLiveWaitSecret loads one persisted runtime secret from disk.
func readRuntimeLiveWaitSecret(secretPath string) (string, error) {
	data, err := os.ReadFile(secretPath)
	if err != nil {
		return "", err
	}
	secret := strings.TrimSpace(string(data))
	if secret == "" {
		return "", fmt.Errorf("%w: %q is empty", errRuntimeLiveWaitSecretInvalid, secretPath)
	}
	if len(secret) != 64 {
		return "", fmt.Errorf("%w: %q has unexpected length", errRuntimeLiveWaitSecretInvalid, secretPath)
	}
	if _, err := hex.DecodeString(secret); err != nil {
		return "", fmt.Errorf("%w: %q decode error: %v", errRuntimeLiveWaitSecretInvalid, secretPath, err)
	}
	return secret, nil
}

// writeRuntimeLiveWaitSecret persists one runtime secret atomically using a temp file and exclusive link.
func writeRuntimeLiveWaitSecret(secretPath, secret string) error {
	tmpFile, err := os.CreateTemp(filepath.Dir(secretPath), runtimeLiveWaitSecretFilename+".tmp-*")
	if err != nil {
		return fmt.Errorf("create live wait secret temp file: %w", err)
	}
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}()
	if err := tmpFile.Chmod(0o600); err != nil {
		return fmt.Errorf("chmod live wait secret temp file %q: %w", tmpFile.Name(), err)
	}
	if _, err := tmpFile.WriteString(secret + "\n"); err != nil {
		return fmt.Errorf("write live wait secret temp file %q: %w", tmpFile.Name(), err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close live wait secret temp file %q: %w", tmpFile.Name(), err)
	}
	if err := os.Link(tmpFile.Name(), secretPath); err != nil {
		if errors.Is(err, os.ErrExist) {
			return os.ErrExist
		}
		return fmt.Errorf("link live wait secret temp file into place: %w", err)
	}
	return nil
}

// generateRuntimeLiveWaitSecret returns one new random runtime secret.
func generateRuntimeLiveWaitSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate live wait secret: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
