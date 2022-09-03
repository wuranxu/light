package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/wuranxu/light/internal/auth"
	"github.com/wuranxu/light/internal/rpc"
	"net/http"
	"strings"
)

const (
	SignKey      = "pityToken"
	AuthFailCode = 103
)

func GetUserInfo(ctx *gin.Context) (*auth.UserInfo, error) {
	token := ctx.GetHeader("token")
	if s := strings.Split(token, " "); len(s) == 2 {
		token = s[1]
	}
	j := auth.NewJWT(SignKey)
	parseToken, err := j.ParseToken(token)
	if err != nil {
		return nil, err
	}
	return &parseToken.UserInfo, nil
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
