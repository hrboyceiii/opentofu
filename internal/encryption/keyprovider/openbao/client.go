package openbao

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	openbao "github.com/openbao/openbao/api"
)

type client interface {
	WriteWithContext(ctx context.Context, path string, data map[string]interface{}) (*openbao.Secret, error)
}

// service implements missing utility functions from openbao/api such as routing and serialization.
type service struct {
	c client
}

type dataKey struct {
	Plaintext  []byte
	Ciphertext []byte
}

func (s service) generateDataKey(ctx context.Context, keyName string, bitSize int) (dataKey, error) {
	path := fmt.Sprintf("/transit/datakey/plaintext/%s", keyName)

	secret, err := s.c.WriteWithContext(ctx, path, map[string]interface{}{
		"bits": bitSize,
	})
	if err != nil {
		return dataKey{}, fmt.Errorf("sending datakey request to openbao: %w", err)
	}

	key := dataKey{}

	key.Ciphertext, err = retrieveCiphertext(secret)
	if err != nil {
		return dataKey{}, err
	}

	key.Plaintext, err = retrievePlaintext(secret)
	if err != nil {
		return dataKey{}, err
	}

	return key, nil
}

func (s service) decryptData(ctx context.Context, keyName string, ciphertext []byte) ([]byte, error) {
	path := fmt.Sprintf("/transit/decrypt/%s", keyName)

	secret, err := s.c.WriteWithContext(ctx, path, map[string]interface{}{
		"ciphertext": string(ciphertext),
	})
	if err != nil {
		return nil, fmt.Errorf("sending decryption request to openbao: %w", err)
	}

	return retrievePlaintext(secret)
}

func retrievePlaintext(s *openbao.Secret) ([]byte, error) {
	base64Plaintext, ok := s.Data["plaintext"].(string)
	if !ok {
		return nil, errors.New("failed to deserialize 'plaintext' from openbao")
	}

	plaintext, err := base64.StdEncoding.DecodeString(base64Plaintext)
	if err != nil {
		return nil, fmt.Errorf("base64 decoding 'plaintext' from openbao: %w", err)
	}

	return plaintext, nil
}

func retrieveCiphertext(s *openbao.Secret) ([]byte, error) {
	ciphertext, ok := s.Data["ciphertext"].(string)
	if !ok {
		return nil, errors.New("failed to deserialize 'ciphertext' from openbao")
	}

	return []byte(ciphertext), nil
}
