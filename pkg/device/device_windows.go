//go:build windows

package device

import (
	"golang.org/x/sys/windows/registry"
)

// Collect 采集当前设备信息
func Collect() Info {
	return Info{
		ID:      regValue(`SOFTWARE\Microsoft\Cryptography`, "MachineGuid"),
		Version: windowsVersion(),
		Model:   regValue(`HARDWARE\DESCRIPTION\System\BIOS`, "SystemProductName"),
	}
}

// displayName 无平台专属显示名，回退到 os.Hostname()
func displayName() string {
	return ""
}

// windowsVersion 返回形如 "Windows 11 Pro 23H2" 的系统版本
func windowsVersion() string {
	const keyPath = `SOFTWARE\Microsoft\Windows NT\CurrentVersion`
	product := regValue(keyPath, "ProductName")
	release := regValue(keyPath, "DisplayVersion")
	if release == "" {
		release = regValue(keyPath, "CurrentBuild")
	}
	if release == "" {
		return product
	}
	return product + " " + release
}

func regValue(path, name string) string {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer key.Close()
	val, _, err := key.GetStringValue(name)
	if err != nil {
		return ""
	}
	return val
}
