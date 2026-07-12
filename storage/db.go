package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Node struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	RawLink  string `json:"raw_link"`
	Group    string `json:"group"` 
	Ping     string `json:"ping"` 
}


type Subscription struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}


type DB struct {
	Subscriptions []Subscription `json:"subscriptions"`
	Nodes         []Node         `json:"nodes"`
}


func getDBPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "." 
	}
	appDir := filepath.Join(configDir, "xray-cli")
	os.MkdirAll(appDir, 0755) 
	return filepath.Join(appDir, "db.json")
}


func LoadDB() (*DB, error) {
	dbPath := getDBPath()
	
	
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


func SaveDB(db *DB) error {
	dbPath := getDBPath()
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dbPath, data, 0644)
}
