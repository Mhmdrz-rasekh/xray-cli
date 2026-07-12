package core

import (
	"os"
	"os/exec"
	"strings"
)

// EnableSystemProxy configures the system to use the local proxy (GNOME/KDE)
func EnableSystemProxy() {
	de := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))

	if strings.Contains(de, "gnome") || strings.Contains(de, "unity") || strings.Contains(de, "pantheon") {
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "manual").Run()
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "host", "127.0.0.1").Run()
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "port", "10808").Run()
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "host", "127.0.0.1").Run()
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "port", "10809").Run()
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "host", "127.0.0.1").Run()
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "port", "10809").Run()
	} else if strings.Contains(de, "kde") {
		_ = exec.Command("kwriteconfig5", "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "ProxyType", "1").Run()
		_ = exec.Command("kwriteconfig5", "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "httpProxy", "http://127.0.0.1:10809").Run()
		_ = exec.Command("kwriteconfig5", "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "httpsProxy", "http://127.0.0.1:10809").Run()
		_ = exec.Command("kwriteconfig5", "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "socksProxy", "socks://127.0.0.1:10808").Run()
		
		// Reload KIO daemon to apply settings
		_ = exec.Command("dbus-send", "--type=signal", "/KIO/Scheduler", "org.kde.KIO.Scheduler.reparseSlaveConfiguration", "string:''").Run()
	}
}

// DisableSystemProxy removes the system proxy settings
func DisableSystemProxy() {
	de := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))

	if strings.Contains(de, "gnome") || strings.Contains(de, "unity") || strings.Contains(de, "pantheon") {
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none").Run()
	} else if strings.Contains(de, "kde") {
		_ = exec.Command("kwriteconfig5", "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "ProxyType", "0").Run()
		_ = exec.Command("dbus-send", "--type=signal", "/KIO/Scheduler", "org.kde.KIO.Scheduler.reparseSlaveConfiguration", "string:''").Run()
	}
}
