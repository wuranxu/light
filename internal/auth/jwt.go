package auth

import (
	"encoding/json"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"time"
)

var (
	TokenExpired     = errors.New("身份认证已过期, 请重新登录")
	TokenNotValidYet = errors.New("身份认证未生效")
	TokenMalformed   = errors.New("登录状态已失效, 请重新登录")
	TokenInvalid     = errors.New("身份认证不合法")
)

type UserInfo struct {
	ID    int    `json:"id"`    // userId
	Email string `json:"email"` // user email
	Name  string `json:"name"`  // username
	Role  int    `json:"role"`
}

type CustomClaims struct {
	UserInfo
	jwt.StandardClaims
}

func (c *UserInfo) Marshal() []byte {
	b, _ := json.Marshal(c)
	return b
}

type JWT struct {
	signKey []byte
}

func NewJWT(key string) *JWT {
	return &JWT{[]byte(key)}
}

// GetSignKey get the sign key
func (j *JWT) GetSignKey() string {
	return string(j.signKey)
}

// SetSignKey set sign key for jwt
func (j *JWT) SetSignKey(key string) {
	j.signKey = []byte(key)
}

func (j *JWT) CreateToken(claims CustomClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.signKey)
}

func (j *JWT) ParseToken(t string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(t, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.signKey, nil
	})
	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return nil, TokenMalformed
			} else if ve.Errors&jwt.ValidationErrorExpired != 0 {
				// Token is expired
				return nil, TokenExpired
			} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				return nil, TokenNotValidYet
			} else {
				return nil, TokenInvalid
			}
		}
		return nil, err
	}
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, TokenInvalid
}

func (j *JWT) RefreshToken(tokenStr string) (string, error) {
	jwt.TimeFunc = func() time.Time {
		return time.Unix(0, 0)
	}
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.signKey, nil
	})
	if err != nil {
		return "", err
	}
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		jwt.TimeFunc = time.Now
		claims.StandardClaims.ExpiresAt = time.Now().Add(1 * time.Hour).Unix()
		return j.CreateToken(*claims)
	}
	return "", TokenInvalid
}
