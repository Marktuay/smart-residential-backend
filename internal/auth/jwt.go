package auth

import (
	"errors"
	"fmt"
	"os"
	"smart-residential-backend/internal/model"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTSecretKey es la clave para firmar los tokens (cargada desde JWT_SECRET)
var JWTSecretKey []byte

// InitJWT inicializa y valida la clave criptográfica de firma JWT desde el entorno
func InitJWT() error {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		return errors.New("la variable de entorno JWT_SECRET es obligatoria y debe tener al menos 32 caracteres")
	}
	JWTSecretKey = []byte(secret)
	return nil
}

// Claims define la estructura del payload del JWT
type Claims struct {
	UsuarioID     int64      `json:"usuario_id"`
	ResidencialID string     `json:"residencial_id"`
	Rol           model.Role `json:"rol"`
	jwt.RegisteredClaims
}

// GenerarToken crea un nuevo JWT para el usuario autenticado
func GenerarToken(usuarioID int64, residencialID string, rol model.Role) (string, error) {
	if len(JWTSecretKey) == 0 {
		return "", errors.New("JWT_SECRET no inicializado")
	}
	claims := &Claims{
		UsuarioID:     usuarioID,
		ResidencialID: residencialID,
		Rol:           rol,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecretKey)
}

// ValidarToken verifica y extrae los claims de un JWT
func ValidarToken(tokenString string) (*Claims, error) {
	if len(JWTSecretKey) == 0 {
		return nil, errors.New("JWT_SECRET no inicializado")
	}
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de firma inesperado: %v", token.Header["alg"])
		}
		return JWTSecretKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("token inválido")
	}

	return claims, nil
}
