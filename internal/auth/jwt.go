package auth

import (
	"errors"
	"smart-residential-backend/internal/model"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTSecretKey es la clave para firmar los tokens (en producción debe venir de una variable de entorno)
var JWTSecretKey = []byte("super-secret-key-para-sagr360")

// Claims define la estructura del payload del JWT
type Claims struct {
	UsuarioID     int64      `json:"usuario_id"`
	ResidencialID string     `json:"residencial_id"`
	Rol           model.Role `json:"rol"`
	jwt.RegisteredClaims
}

// GenerarToken crea un nuevo JWT para el usuario autenticado
func GenerarToken(usuarioID int64, residencialID string, rol model.Role) (string, error) {
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
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
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
