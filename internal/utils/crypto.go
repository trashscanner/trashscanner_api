package utils

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"
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

	pemPrivateKey, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		log.Fatalf("failed to marshal private key: %v", err)
	}
	privateKeyPEM := pem.EncodeToMemory(pemPrivateKey)
	privateKeyBase64 := base64.StdEncoding.EncodeToString(privateKeyPEM)

	sshPublicKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		log.Fatalf("failed to create SSH public key: %v", err)
	}
	publicKeyBase64 := base64.StdEncoding.EncodeToString(ssh.MarshalAuthorizedKey(sshPublicKey))

	if err := os.Setenv("AUTH_MANAGER_SECRET_PRIVATE_KEY", privateKeyBase64); err != nil {
		log.Fatalf("failed to set private key env: %v", err)
	}
	if err := os.Setenv("AUTH_MANAGER_PUBLIC_KEY", publicKeyBase64); err != nil {
		log.Fatalf("failed to set public key env: %v", err)
	}
}

func GetEdDSAKeysFromEnv() (ed25519.PrivateKey, ed25519.PublicKey, error) {
	privateKeyBase64 := os.Getenv("AUTH_MANAGER_SECRET_PRIVATE_KEY")
	publicKeyBase64 := os.Getenv("AUTH_MANAGER_PUBLIC_KEY")

	if privateKeyBase64 == "" || publicKeyBase64 == "" {
		return nil, nil, fmt.Errorf("EdDSA keys not found in environment variables")
	}

	privateKeyPEM, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	publicKeyPEM, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	privateKey, err := ssh.ParseRawPrivateKey(privateKeyPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	ed25519PrivateKey, ok := privateKey.(*ed25519.PrivateKey)
	if !ok {
		return nil, nil, fmt.Errorf("private key is not Ed25519 type")
	}

	sshPublicKey, _, _, _, err := ssh.ParseAuthorizedKey(publicKeyPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	cryptoPublicKey := sshPublicKey.(ssh.CryptoPublicKey).CryptoPublicKey()

	ed25519PublicKey, ok := cryptoPublicKey.(ed25519.PublicKey)
	if !ok {
		return nil, nil, fmt.Errorf("public key is not Ed25519 type")
	}

	return *ed25519PrivateKey, ed25519PublicKey, nil
}
