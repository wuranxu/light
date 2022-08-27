package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/wuranxu/light/internal/auth"
	"github.com/wuranxu/light/internal/rpc"
	"net/http"
	"strings"
)

const (
	SignKey      = "PITY_GATEWAY"
	AuthFailCode = 103
)

func GetUserInfo(ctx *gin.Context) (*auth.CustomClaims, error) {
	token := ctx.GetHeader("Authorization")
	if s := strings.Split(token, " "); len(s) == 2 {
		token = s[1]
	}
	j := auth.NewJWT(SignKey)
	return j.ParseToken(token)
}

func Auth(ctx *gin.Context) {
	claims, err := GetUserInfo(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, rpc.Response{Code: AuthFailCode, Msg: err.Error()})
		ctx.Abort()
		return
	}
	ctx.Set("userInfo", claims)
	ctx.Next()
}
