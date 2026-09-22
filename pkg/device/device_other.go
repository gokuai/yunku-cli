//go:build !darwin && !linux && !windows

package device

import "runtime"

// Collect 采集当前设备信息
func Collect() Info {
	return Info{Version: runtime.GOOS}
}

// displayName 无平台专属显示名，回退到 os.Hostname()
func displayName() string {
	return ""
}
