package common

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

//claim represents the data stores in JWT token
type Claims struct {
	// user id and hadle are custom claim , 
	UserID uint64 `json:"user_id"`
	Handle string `json:"handle"` //custom claim
	jwt.RegisteredClaims //it comes from package like exp, etc
}

func GenerateToken(userID uint64, handle string) (string, error) {
	claims := &Claims{
		UserID: userID,
		Handle: handle,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt	: jwt.NewNumericDate(time.Now().Add(24*time.Hour)),
			IssuedAt	: jwt.NewNumericDate(time.Now()),
			Issuer		: "gosocial",
			Subject		: "user-auth",	
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(jwtSecret)
}


func ValidToken(tokenstring string) (*Claims, error){
	token , err := jwt.ParseWithClaims(tokenstring, &Claims{}, func(token *jwt.Token)(interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok{
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}




