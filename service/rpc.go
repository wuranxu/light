package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/wuranxu/light/internal/auth"
	"github.com/wuranxu/light/internal/rpc"
	"github.com/wuranxu/light/middleware"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

const (
	// ArgsParseFailed 参数解析失败 10001
	ArgsParseFailed = 10001 + iota
	// LoginRequired 用户未登录 10002
	LoginRequired
	// MethodNotFound 方法未找到 10003
	MethodNotFound
	// RemoteCallFailed rpc调用失败 10004
	RemoteCallFailed
	// IntervalServerError 服务出错 10005
	IntervalServerError
)

var (
	InnerError  = errors.New("系统内部错误")
	SystemError = errors.New("抱歉, 网络似乎开小差了")

	// 服务连接缓存
	ClientCache = &GrpcCache{cache: make(map[string]*rpc.GrpcClient)}
)

type GrpcCache struct {
	lock  sync.RWMutex
	cache map[string]*rpc.GrpcClient
}

func (g *GrpcCache) GetClient(service string) (*rpc.GrpcClient, error) {
	g.lock.RLock()
	client, ok := g.cache[service]
	g.lock.RUnlock()
	if ok {
		return client, nil
	}
	client, err := rpc.NewGrpcClient(service)
	if err != nil {
		return nil, err
	}
	g.SetClient(service, client)
	return client, nil
}

func (g *GrpcCache) SetClient(service string, client *rpc.GrpcClient) {
	g.lock.Lock()
	defer g.lock.Unlock()
	g.cache[service] = client
}

type Response interface {
	toJson() []byte
}

type res struct {
	Code int32       `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func (s *res) Build(code int32, msg interface{}, data ...interface{}) *res {
	s.SetMsg(msg).Code = code
	if len(data) > 0 {
		s.Data = data[0]
	}
	return s
}

func (s *res) SetMsg(msg interface{}) *res {
	switch msg.(type) {
	case string:
		s.Msg = msg.(string)
	default:
		s.Msg = fmt.Sprintf("%v", msg)
	}
	return s
}

func (s *res) toApi(resp *rpc.Response) *res {
	if resp.ResultJson != nil {
		if err := json.Unmarshal(resp.ResultJson, &s.Data); err != nil {
			return s.Build(IntervalServerError, InnerError)
		}
	}
	return s.Build(resp.Code, resp.Msg)
}

type Params map[string]interface{}

func (p Params) Marshal() (*rpc.Request, error) {
	marshal, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return &rpc.Request{RequestJson: marshal}, nil
}

func (p Params) MakeFile(ctx *gin.Context) {
	list := fileNameList(ctx)
	if list == nil {
		return
	}
	fileList := make([]map[string]interface{}, 0, len(list))
	for _, l := range list {
		file, err := ctx.FormFile(l)
		if err != nil {
			continue
		}
		open, err := file.Open()
		if err != nil {
			continue
		}
		buf, err := ioutil.ReadAll(open)
		if err != nil {
			continue
		}
		if err = open.Close(); err != nil {
			continue
		}
		fileList = append(fileList, map[string]interface{}{
			"filename": file.Filename,
			"size":     file.Size,
			"content":  buf,
		})
	}
	p["fileList"] = fileList
}

func response(ctx *gin.Context, r *res) {
	ctx.JSON(http.StatusOK, r)
}

func fileNameList(ctx *gin.Context) []string {
	fileList := ctx.Query("files")
	if fileList == "" {
		return nil
	}
	return strings.Split(fileList, ";")
}

// CallRpc rpc调用接口
func CallRpc(ctx *gin.Context) {
	result := new(res)
	params := make(Params)
	var (
		userInfo    *auth.CustomClaims
		requestData *rpc.Request
		err         error
	)
	// 如果是form
	if strings.Contains(ctx.GetHeader("Content-Type"), "form") {
		values := ctx.Request.PostForm
		params.MakeFile(ctx)
		for k, v := range values {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
		requestData, err = params.Marshal()
		if err != nil {
			response(ctx, result.Build(ArgsParseFailed, err))
			return
		}
	} else {
		request, err := ioutil.ReadAll(ctx.Request.Body)
		if err != nil {
			response(ctx, result.Build(ArgsParseFailed, SystemError))
			return
		}
		requestData = &rpc.Request{RequestJson: request}
	}
	// 获取url中版本/APP/方法名(首字母小写, 与其他语言服务保持一致)
	version := ctx.Param("version")
	service := ctx.Param("service")
	method := ctx.Param("method")
	client, err := ClientCache.GetClient(service)
	if err != nil {
		response(ctx, result.Build(MethodNotFound, err))
		return
	}
	addr, err := client.SearchCallAddr(version, service, method)
	if err != nil {
		response(ctx, result.Build(MethodNotFound, err))
		return
	}
	if addr.Authorization {
		// 需要解析token
		if userInfo, err = middleware.GetUserInfo(ctx); err != nil {
			response(ctx, result.Build(LoginRequired, err))
			return
		}
	}
	resp, err := client.Invoke(addr, requestData, ctx.RemoteIP(), userInfo)
	if err != nil {
		response(ctx, result.Build(RemoteCallFailed, err))
		return
	}
	response(ctx, result.toApi(resp))
}
