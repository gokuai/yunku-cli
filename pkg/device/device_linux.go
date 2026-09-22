//go:build linux

package device

import (
	"os"
	"strings"
)

// Collect 采集当前设备信息
func Collect() Info {
	return Info{
		ID:      machineID(),
		Version: osPrettyName(),
		Model:   readTrimmed("/sys/class/dmi/id/product_name"),
	}
}

// displayName 无平台专属显示名，回退到 os.Hostname()
func displayName() string {
	return ""
}

// machineID 返回 /etc/machine-id 中的系统唯一标识
func machineID() string {
	for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		if id := readTrimmed(path); id != "" {
			return id
		}
	}
	return ""
}

// osPrettyName 返回 /etc/os-release 中的 PRETTY_NAME
func osPrettyName() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if k, v, ok := strings.Cut(line, "="); ok && k == "PRETTY_NAME" {
			return strings.Trim(strings.TrimSpace(v), `"`)
		}
	}
	return ""
}

func readTrimmed(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
