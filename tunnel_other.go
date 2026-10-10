//go:build !windows

package main

import "errors"

func tunnelConnect(a *App, id, key string) error { return errors.New("disponivel apenas no Windows") }
func tunnelDisconnect(a *App) error              { return errors.New("disponivel apenas no Windows") }
func tunnelAutoStart(a *App)                     {}
func tunnelStatus(a *App) map[string]any {
	return map[string]any{"installed": false, "configured": false, "status": "incompativel"}
}
