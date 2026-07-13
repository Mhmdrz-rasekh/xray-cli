package core

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/Mhmdrz-rasekh/xray-cli/parser"
)

// MeasureRealPing starts a temporary Xray instance and tests actual HTTP ping
func MeasureRealPing(node *parser.VlessNode, xrayPath string) (time.Duration, error) {
	testPort := 20808 // پورت موقت برای تست

	// تولید کانفیگ برای سرور جهت تست
	cfgPath, err := GenerateConfig(node, "manual", testPort)
	if err != nil {
		return 0, err
	}
	// پاک کردن فایل کانفیگ پس از اتمام تست
	defer os.Remove(cfgPath)

	// اجرای هسته Xray به صورت مخفی در پس‌زمینه
	cmd := exec.Command(xrayPath, "run", "-c", cfgPath)
	if err := cmd.Start(); err != nil {
		return 0, err
	}

	// اطمینان از بسته شدن پروسه هسته بعد از پایان تست
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// نیم ثانیه مکث تا هسته بالا بیاید و پورت باز شود
	time.Sleep(500 * time.Millisecond)

	// تنظیم کلاینت HTTP برای عبور از پروکسی SOCKS5 ایجاد شده
	proxyURL, _ := url.Parse(fmt.Sprintf("socks5://127.0.0.1:%d", testPort))
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 3 * time.Second, // تایم‌اوت مشخص برای جلوگیری از گیر کردن
	}

	start := time.Now()
	req, err := http.NewRequest("GET", "https://www.gstatic.com/generate_204", nil)
	if err != nil {
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// اطمینان از اینکه واقعا به اینترنت آزاد وصل شده‌ایم
	if resp.StatusCode != 204 {
		return 0, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return time.Since(start), nil
}
