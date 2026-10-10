//go:build !windows

package main

import (
	"errors"
	"time"
)

type psProcess struct {
	Name string  `json:"ProcessName"`
	PID  int     `json:"Id"`
	RAM  float64 `json:"RamMB"`
}

var closable = map[string]bool{"chrome": true, "msedge": true, "discord": true}

func systemMetrics() (map[string]any, error) {
	return map[string]any{"platform": "test-only", "total_ram_bytes": 0, "available_ram_bytes": 0, "used_ram_bytes": 0, "cpu_percent": 0}, nil
}
func processList() ([]psProcess, error)             { return []psProcess{}, nil }
func processName(pid int) (string, error)           { return "", errors.New("Windows only") }
func validateProcess(p ProcessTarget) error         { return errors.New("Windows only") }
func closeProcess(p ProcessTarget) error            { return errors.New("Windows only") }
func activePower() (string, error)                  { return "", errors.New("Windows only") }
func highPower() (string, error)                    { return "", errors.New("Windows only") }
func activatePower(s string) error                  { return errors.New("Windows only") }
func boosterSession() map[string]any                { return map[string]any{"external_booster_session": false} }
func restrictedCommand(a, d string) (string, error) { return "", errors.New("Windows only") }
func openDashboard(s string)                        {}

var _ = time.Second
