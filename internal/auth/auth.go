package auth

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Martin-Hayot/auction-server/pkg/errors"
	"github.com/charmbracelet/log"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwe"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"golang.org/x/crypto/hkdf"
)

func GenerateEncryptionKey() ([]byte, error) {
	authSecret := os.Getenv("AUTH_SECRET")
	if authSecret == "" {
		return nil, errors.New(500, "AUTH_SECRET not set")
	}

	salt := "authjs.session-token"
	info := fmt.Sprintf("Auth.js Generated Encryption Key (%s)", salt)

	// HKDF with SHA-256
	hash := sha256.New
	kdf := hkdf.New(hash, []byte(authSecret), []byte(salt), []byte(info))

	key := make([]byte, 64)
	if _, err := io.ReadFull(kdf, key); err != nil {
		return nil, errors.Wrap(err, "failed to generate key")
	}

	return key, nil
}

func JweToJwt(encryptedToken string) (string, error) {
	key, err := GenerateEncryptionKey()
	if err != nil {
		return "", errors.Wrap(err, "failed to generate encryption key")
	}

	// Decrypt JWE using DIRECT key encryption and A256GCM content encryption
	decrypted, err := jwe.Decrypt([]byte(encryptedToken),
		jwe.WithKey(jwa.DIRECT(), key))
	if err != nil {
		return "", errors.Wrap(err, "failed to decrypt JWE")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(decrypted, &payload); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal decrypted payload")
	}

	token := jwt.New()
	for k, v := range payload {
		token.Set(k, v)
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256(), []byte(os.Getenv("AUTH_SECRET"))))
	if err != nil {
		return "", errors.Wrap(err, "failed to sign JWT")
	}

	return string(signed), nil
}

func ValidateTokenFromCookie(r *http.Request) (jwt.Token, error) {
	cookie, err := r.Cookie("authjs.session-token")
	if err != nil {
		return nil, errors.New(http.StatusUnauthorized, "missing session token cookie")
	}

	// Convert JWE to JWT
	jwtString, err := JweToJwt(cookie.Value)
	if err != nil {
		log.Error("Failed to convert JWE to JWT", "error", err)
		return nil, errors.Wrap(err, "failed to convert JWE to JWT")
	}

	// Verify JWT
	token, err := jwt.Parse([]byte(jwtString),
		jwt.WithKey(jwa.HS256(), []byte(os.Getenv("AUTH_SECRET"))),
		jwt.WithValidate(true))
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate token")
	}

	// Check expiration
	if exp, ok := token.Expiration(); ok && exp.Before(time.Now()) {
		return nil, errors.New(http.StatusUnauthorized, "session token expired")
	}

	return token, nil
}
