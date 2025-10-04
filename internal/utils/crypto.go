package utils

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func HashPass(pass string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

func CompareHashPass(hashedPass, pass string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPass), []byte(pass))
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func CompareTokenHash(storedHash, token string) bool {
	tokenHash := HashToken(token)
	return storedHash == tokenHash
}

func GenerateAndSetKeys() {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("failed to generate EdDSA keys: %v", err)
	}

	privateKeyBase64 := base64.StdEncoding.EncodeToString(privateKey)
	publicKeyBase64 := base64.StdEncoding.EncodeToString(publicKey)

	os.Setenv("AUTH_MANAGER_SECRET_PRIVATE_KEY", privateKeyBase64)
	os.Setenv("AUTH_MANAGER_PUBLIC_KEY", publicKeyBase64)
}

func GetEdDSAKeysFromEnv() (ed25519.PrivateKey, ed25519.PublicKey, error) {
	privateKeyBase64 := os.Getenv("AUTH_MANAGER_SECRET_PRIVATE_KEY")
	publicKeyBase64 := os.Getenv("AUTH_MANAGER_PUBLIC_KEY")

	if privateKeyBase64 == "" || publicKeyBase64 == "" {
		return nil, nil, fmt.Errorf("EdDSA keys not found in environment variables")
	}

	privateKey, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	publicKey, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	return ed25519.PrivateKey(privateKey), ed25519.PublicKey(publicKey), nil
}
