package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Node ساختار یک کانفیگ (گره) را مشخص می‌کند
type Node struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	RawLink  string `json:"raw_link"`
	Group    string `json:"group"` // اسم ساب‌اسکریپشن یا "Local"
	Ping     string `json:"ping"`   // آخرین وضعیت پینگ
}

// Subscription ساختار یک لینک اشتراک را مشخص می‌کند
type Subscription struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// DB ساختار اصلی دیتابیس لوکال ماست
type DB struct {
	Subscriptions []Subscription `json:"subscriptions"`
	Nodes         []Node         `json:"nodes"`
}

// getDBPath مسیر فایل دیتابیس را پیدا کرده و در صورت نیاز پوشه‌های آن را می‌سازد
func getDBPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "." // اگر پیدا نشد، در همان پوشه فعلی بساز
	}
	appDir := filepath.Join(configDir, "xray-cli")
	os.MkdirAll(appDir, 0755) // ساخت پوشه در صورت عدم وجود
	return filepath.Join(appDir, "db.json")
}

// LoadDB فایل دیتابیس را می‌خواند و در قالب ساختار DB برمی‌گرداند
func LoadDB() (*DB, error) {
	dbPath := getDBPath()
	
	// اگر فایل وجود نداشت، یک دیتابیس خالی برگردان
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return &DB{
			Subscriptions: []Subscription{},
			Nodes:         []Node{},
		}, nil
	}

	data, err := os.ReadFile(dbPath)
	if err != nil {
		return nil, err
	}

	var db DB
	if err := json.Unmarshal(data, &db); err != nil {
		return nil, err
	}

	return &db, nil
}

// SaveDB اطلاعات جدید را روی فایل دیتابیس می‌نویسد (ذخیره می‌کند)
func SaveDB(db *DB) error {
	dbPath := getDBPath()
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dbPath, data, 0644)
}
