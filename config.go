package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config is saved in ~/Library/Application Support/BTCMenuBar/config.json.
type Config struct {
	Currency        string  `json:"currency"`
	IntervalSeconds int     `json:"interval_seconds"`
	ShowChange      bool    `json:"show_change"`
	AutoCheckUpdate bool    `json:"auto_check_update"`
	FontSize        float64 `json:"font_size"` // points; 0 = the normal macOS menu bar size
}

func defaultConfig() Config {
	return Config{Currency: "usd", IntervalSeconds: 60, ShowChange: true, AutoCheckUpdate: true, FontSize: 16}
}

// text sizes offered in the menu: label and points (0 = normal macOS size)
var fontSizes = []struct {
	Label string
	Pt    float64
}{{"Normal", 0}, {"Large", 16}, {"Extra Large", 18}}

var intervals = []time.Duration{30 * time.Second, time.Minute, 5 * time.Minute, 15 * time.Minute}

func configPath() (string, error) {
	dir, err := os.UserConfigDir() // ~/Library/Application Support on macOS
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "BTCMenuBar", "config.json"), nil
}

func loadConfig() Config {
	c := defaultConfig()
	p, err := configPath()
	if err != nil {
		return c
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return c
	}
	_ = json.Unmarshal(b, &c)
	if _, ok := currencySymbols[c.Currency]; !ok {
		c.Currency = "usd"
	}
	if c.IntervalSeconds < 30 {
		c.IntervalSeconds = 30
	}
	return c
}

func saveConfig(c Config) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(p, b, 0o644)
}

/* ---------- Launch at login (a LaunchAgent in ~/Library/LaunchAgents) ---------- */

const agentLabel = "com.jamalxcode.btconmenubar"

func agentPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", agentLabel+".plist")
}

func launchAtLoginEnabled() bool {
	_, err := os.Stat(agentPath())
	return err == nil
}

func setLaunchAtLogin(on bool) error {
	if !on {
		err := os.Remove(agentPath())
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	app, err := bundlePath()
	if err != nil {
		return err
	}
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>%s</string>
  <key>ProgramArguments</key>
  <array><string>/usr/bin/open</string><string>-a</string><string>%s</string></array>
  <key>RunAtLoad</key><true/>
</dict>
</plist>
`, agentLabel, xmlEscape(app))
	if err := os.MkdirAll(filepath.Dir(agentPath()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(agentPath(), []byte(plist), 0o644)
}

func xmlEscape(s string) string {
	r := []rune{}
	for _, c := range s {
		switch c {
		case '&':
			r = append(r, []rune("&amp;")...)
		case '<':
			r = append(r, []rune("&lt;")...)
		case '>':
			r = append(r, []rune("&gt;")...)
		default:
			r = append(r, c)
		}
	}
	return string(r)
}
