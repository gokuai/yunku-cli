//go:build darwin

package device

import (
	"os/exec"
	"strings"
)

// Collect 采集当前设备信息
func Collect() Info {
	return Info{
		ID:      hardwareUUID(),
		Version: osVersion(),
		Model:   sysctlValue("hw.model"),
	}
}

// hardwareUUID 返回 Mac 的 IOPlatformUUID
func hardwareUUID() string {
	out, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "IOPlatformUUID") {
			continue
		}
		// 格式: "IOPlatformUUID" = "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
		parts := strings.Split(line, `"`)
		if len(parts) >= 4 {
			return parts[len(parts)-2]
		}
	}
	return ""
}

// osVersion 返回形如 "Mac OS X 14.5" 的系统版本（与桌面端 systeminformation 的 distro 命名一致）
func osVersion() string {
	out, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		return ""
	}
	version := strings.TrimSpace(string(out))
	if version == "" {
		return ""
	}
	return "Mac OS X " + version
}

// displayName 返回用户可读的电脑名称（如 "林盛的MacBook Pro"）
func displayName() string {
	out, err := exec.Command("scutil", "--get", "ComputerName").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func sysctlValue(key string) string {
	out, err := exec.Command("sysctl", "-n", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
