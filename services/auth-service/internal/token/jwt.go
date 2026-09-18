package token

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/config"
	"go.uber.org/zap"
)

type ScopeClaim struct {
	Role string `json:"role"`
	Type string `json:"type"`
	ID   string `json:"id"`
}

type AccessClaims struct {
	Typ    string       `json:"typ"`
	Email  string       `json:"email"`
	Name   string       `json:"name"`
	Roles  []string     `json:"roles"`
	Scopes []ScopeClaim `json:"scopes"`
	Sid    string       `json:"sid"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	Typ string `json:"typ"`
	Sid string `json:"sid"`
	jwt.RegisteredClaims
}

type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

type TokenManager interface {
	GenerateAccessToken(userID uuid.UUID, email, name string, roles []string, scopes []ScopeClaim, sessionID string) (tokenStr string, jti string, expiresAt time.Time, err error)
	GenerateRefreshToken(userID uuid.UUID, sessionID string) (tokenStr string, jti string, expiresAt time.Time, err error)
	ValidateAccessToken(tokenStr string) (*AccessClaims, error)
	ValidateRefreshToken(tokenStr string) (*RefreshClaims, error)
	GetJWKS() JWKSResponse
	PublicKey() *rsa.PublicKey
}

type tokenManager struct {
	cfg        *config.Config
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	log        *zap.Logger
}

func NewTokenManager(cfg *config.Config, log *zap.Logger) (TokenManager, error) {
	var privateKey *rsa.PrivateKey
	var err error

	if cfg.JWTPrivateKeyPath != "" {
		privateKey, err = loadPrivateKeyFromFile(cfg.JWTPrivateKeyPath)
		if err != nil {
			log.Warn("Gagal memuat private key dari path file", zap.String("path", cfg.JWTPrivateKeyPath), zap.Error(err))
		}
	}

	if privateKey == nil {
		if cfg.AppEnv == "development" || cfg.AppEnv == "test" {
			log.Warn("⚠️ JWT_PRIVATE_KEY_PATH tidak ada/invalid. Meng-generate RSA private key sementara untuk development...")
			privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				return nil, fmt.Errorf("gagal generate RSA key: %w", err)
			}
		} else {
			return nil, errors.New("JWT_PRIVATE_KEY_PATH wajib diatur untuk environment non-development")
		}
	}

	return &tokenManager{
		cfg:        cfg,
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		log:        log,
	}, nil
}

func loadPrivateKeyFromFile(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("format PEM private key tidak valid")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	keyIf, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := keyIf.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("kunci bukan RSA private key")
	}

	return rsaKey, nil
}

func (tm *tokenManager) GenerateAccessToken(
	userID uuid.UUID,
	email, name string,
	roles []string,
	scopes []ScopeClaim,
	sessionID string,
) (string, string, time.Time, error) {
	now := time.Now()
	jti := uuid.NewString()
	exp := now.Add(tm.cfg.JWTAccessTTL)

	if sessionID == "" {
		sessionID = uuid.NewString()
	}

	claims := AccessClaims{
		Typ:    "access",
		Email:  email,
		Name:   name,
		Roles:  roles,
		Scopes: scopes,
		Sid:    sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tm.cfg.JWTIssuer,
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{tm.cfg.JWTAudience},
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = tm.cfg.JWTKid

	signed, err := token.SignedString(tm.privateKey)
	if err != nil {
		return "", "", time.Time{}, err
	}

	return signed, jti, exp, nil
}

func (tm *tokenManager) GenerateRefreshToken(
	userID uuid.UUID,
	sessionID string,
) (string, string, time.Time, error) {
	now := time.Now()
	jti := uuid.NewString()
	exp := now.Add(tm.cfg.JWTRefreshTTL)

	if sessionID == "" {
		sessionID = uuid.NewString()
	}

	claims := RefreshClaims{
		Typ: "refresh",
		Sid: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tm.cfg.JWTIssuer,
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{tm.cfg.JWTAudience},
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = tm.cfg.JWTKid

	signed, err := token.SignedString(tm.privateKey)
	if err != nil {
		return "", "", time.Time{}, err
	}

	return signed, jti, exp, nil
}

func (tm *tokenManager) ValidateAccessToken(tokenStr string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &AccessClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("metode signing token tidak valid: %v", t.Header["alg"])
		}
		return tm.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*AccessClaims)
	if !ok || !token.Valid {
		return nil, errors.New("token tidak valid")
	}

	if claims.Typ != "access" {
		return nil, errors.New("tipe token bukan access token")
	}

	return claims, nil
}

func (tm *tokenManager) ValidateRefreshToken(tokenStr string) (*RefreshClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &RefreshClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("metode signing token tidak valid: %v", t.Header["alg"])
		}
		return tm.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*RefreshClaims)
	if !ok || !token.Valid {
		return nil, errors.New("token tidak valid")
	}

	if claims.Typ != "refresh" {
		return nil, errors.New("tipe token bukan refresh token")
	}

	return claims, nil
}

func (tm *tokenManager) GetJWKS() JWKSResponse {
	nBytes := tm.publicKey.N.Bytes()
	eBytes := big.NewInt(int64(tm.publicKey.E)).Bytes()

	return JWKSResponse{
		Keys: []JWK{
			{
				Kty: "RSA",
				Use: "sig",
				Alg: "RS256",
				Kid: tm.cfg.JWTKid,
				N:   base64.RawURLEncoding.EncodeToString(nBytes),
				E:   base64.RawURLEncoding.EncodeToString(eBytes),
			},
		},
	}
}

func (tm *tokenManager) PublicKey() *rsa.PublicKey {
	return tm.publicKey
}
