package etcd

import (
	"encoding/json"
	"fmt"
	"unicode"
)

type Method struct {
	Authorization bool   `json:"authorization"` // 是否需要登录
	Path          string `json:"path"`
}

func (m *Method) Marshal() string {
	b, _ := json.Marshal(m)
	return string(b)
}

func lowerFirst(str string) string {
	for i, v := range str {
		return string(unicode.ToLower(v)) + str[i+1:]
	}
	return ""
}

func RegisterMethod(client *Client, version, service, method string, auth bool) error {
	md := &Method{
		Authorization: auth,
		Path:          fmt.Sprintf("/%s/%s", service, method),
	}
	fullPath := fmt.Sprintf("%s.%s.%s", version, lowerFirst(service), lowerFirst(method))
	_, err := client.cli.Put(client.cli.Ctx(), fullPath, md.Marshal())
	if err != nil {
		return err
	}
	return nil
}

func UnRegisterMethod(client *Client, version, service, method string) error {
	fullPath := fmt.Sprintf("%s.%s.%s", version, lowerFirst(service), lowerFirst(method))
	_, err := client.cli.Delete(client.cli.Ctx(), fullPath)
	return err
}
