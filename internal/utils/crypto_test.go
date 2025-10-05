package utils

import (
	"crypto/ed25519"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPass_Success(t *testing.T) {
	password := "mySecurePassword123"

	hash, err := HashPass(password)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
	assert.Contains(t, hash, "$2a$")
}

func TestHashPass_EmptyPassword(t *testing.T) {
	hash, err := HashPass("")

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestHashPass_DifferentHashesForSamePassword(t *testing.T) {
	password := "samePassword"

	hash1, err1 := HashPass(password)
	hash2, err2 := HashPass(password)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, hash1, hash2)
}

func TestCompareHashPass_ValidPassword(t *testing.T) {
	password := "correctPassword"
	hash, err := HashPass(password)
	require.NoError(t, err)

	err = CompareHashPass(hash, password)

	assert.NoError(t, err)
}

func TestCompareHashPass_InvalidPassword(t *testing.T) {
	password := "correctPassword"
	hash, err := HashPass(password)
	require.NoError(t, err)

	err = CompareHashPass(hash, "wrongPassword")

	assert.Error(t, err)
}

func TestCompareHashPass_EmptyPassword(t *testing.T) {
	password := "password"
	hash, err := HashPass(password)
	require.NoError(t, err)

	err = CompareHashPass(hash, "")

	assert.Error(t, err)
}

func TestHashToken_Success(t *testing.T) {
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.token"

	hash := HashToken(token)

	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64)
}

func TestHashToken_Deterministic(t *testing.T) {
	token := "sameToken"

	hash1 := HashToken(token)
	hash2 := HashToken(token)

	assert.Equal(t, hash1, hash2)
}

func TestHashToken_DifferentTokens(t *testing.T) {
	token1 := "token1"
	token2 := "token2"

	hash1 := HashToken(token1)
	hash2 := HashToken(token2)

	assert.NotEqual(t, hash1, hash2)
}

func TestHashToken_EmptyToken(t *testing.T) {
	hash := HashToken("")

	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64)
}

func TestCompareTokenHash_Match(t *testing.T) {
	token := "myRefreshToken123"
	hash := HashToken(token)

	result := CompareTokenHash(hash, token)

	assert.True(t, result)
}

func TestCompareTokenHash_NoMatch(t *testing.T) {
	token := "correctToken"
	hash := HashToken(token)

	result := CompareTokenHash(hash, "wrongToken")

	assert.False(t, result)
}

func TestCompareTokenHash_EmptyToken(t *testing.T) {
	token := "token"
	hash := HashToken(token)

	result := CompareTokenHash(hash, "")

	assert.False(t, result)
}

func TestGenerateAndSetKeys_Success(t *testing.T) {
	os.Unsetenv("AUTH_MANAGER_SECRET_PRIVATE_KEY")
	os.Unsetenv("AUTH_MANAGER_PUBLIC_KEY")

	GenerateAndSetKeys()

	privateKeyBase64 := os.Getenv("AUTH_MANAGER_SECRET_PRIVATE_KEY")
	publicKeyBase64 := os.Getenv("AUTH_MANAGER_PUBLIC_KEY")

	assert.NotEmpty(t, privateKeyBase64)
	assert.NotEmpty(t, publicKeyBase64)

	_, _, err := GetEdDSAKeysFromEnv()
	require.NoError(t, err)
}

func TestGenerateAndSetKeys_DifferentKeys(t *testing.T) {
	GenerateAndSetKeys()
	privateKey1 := os.Getenv("AUTH_MANAGER_SECRET_PRIVATE_KEY")
	publicKey1 := os.Getenv("AUTH_MANAGER_PUBLIC_KEY")

	GenerateAndSetKeys()
	privateKey2 := os.Getenv("AUTH_MANAGER_SECRET_PRIVATE_KEY")
	publicKey2 := os.Getenv("AUTH_MANAGER_PUBLIC_KEY")

	assert.NotEqual(t, privateKey1, privateKey2)
	assert.NotEqual(t, publicKey1, publicKey2)
}

func TestGetEdDSAKeysFromEnv_Success(t *testing.T) {
	GenerateAndSetKeys()

	privateKey, publicKey, err := GetEdDSAKeysFromEnv()

	require.NoError(t, err)
	assert.NotNil(t, privateKey)
	assert.NotNil(t, publicKey)
	assert.Len(t, privateKey, ed25519.PrivateKeySize)
	assert.Len(t, publicKey, ed25519.PublicKeySize)
}

func TestGetEdDSAKeysFromEnv_MissingPrivateKey(t *testing.T) {
	os.Setenv("AUTH_MANAGER_PUBLIC_KEY", "dGVzdA==")
	os.Unsetenv("AUTH_MANAGER_SECRET_PRIVATE_KEY")

	_, _, err := GetEdDSAKeysFromEnv()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetEdDSAKeysFromEnv_MissingPublicKey(t *testing.T) {
	os.Setenv("AUTH_MANAGER_SECRET_PRIVATE_KEY", "dGVzdA==")
	os.Unsetenv("AUTH_MANAGER_PUBLIC_KEY")

	_, _, err := GetEdDSAKeysFromEnv()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetEdDSAKeysFromEnv_InvalidPrivateKey(t *testing.T) {
	os.Setenv("AUTH_MANAGER_SECRET_PRIVATE_KEY", "invalid-base64-!!!!")
	os.Setenv("AUTH_MANAGER_PUBLIC_KEY", "dGVzdA==")

	_, _, err := GetEdDSAKeysFromEnv()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode private key")
}

func TestGetEdDSAKeysFromEnv_InvalidPublicKey(t *testing.T) {
	GenerateAndSetKeys()
	os.Setenv("AUTH_MANAGER_PUBLIC_KEY", "invalid-base64-!!!!")

	_, _, err := GetEdDSAKeysFromEnv()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode public key")
}

func TestEdDSAKeys_RoundTrip(t *testing.T) {
	GenerateAndSetKeys()

	privateKey, publicKey, err := GetEdDSAKeysFromEnv()
	require.NoError(t, err)

	message := []byte("test message")
	signature := ed25519.Sign(privateKey, message)

	valid := ed25519.Verify(publicKey, message, signature)
	assert.True(t, valid)
}

func TestPasswordHashing_RoundTrip(t *testing.T) {
	password := "mySecurePassword123"

	hash, err := HashPass(password)
	require.NoError(t, err)

	err = CompareHashPass(hash, password)
	assert.NoError(t, err)

	err = CompareHashPass(hash, "wrongPassword")
	assert.Error(t, err)
}

func TestTokenHashing_RoundTrip(t *testing.T) {
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.refresh.token"

	hash := HashToken(token)

	assert.True(t, CompareTokenHash(hash, token))
	assert.False(t, CompareTokenHash(hash, "wrongToken"))
}
