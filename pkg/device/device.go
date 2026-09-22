// Package device 采集当前设备信息，登录时附带给服务端用于设备识别与管理
package device

import (
	"encoding/json"
	"os"
)

// Info 当前设备信息
type Info struct {
	ID      string // 硬件 UUID
	Version string // 系统发行版 + 版本号
	Model   string // 设备型号
}

// Params 返回登录请求附带的 device 和 info 参数
func Params(clientID string) map[string]string {
	info := Collect()
	display := displayName()
	if display == "" {
		display, _ = os.Hostname()
	}

	infoJSON, _ := json.Marshal(map[string]string{
		"VERSION":   info.Version,
		"BRAND":     info.Model,
		"DISPLAY":   display + "(CLI)",
		"CLIENT_ID": clientID,
	})

	params := map[string]string{"info": string(infoJSON)}
	if info.ID != "" {
		params["device"] = info.ID
	}
	return params
}
