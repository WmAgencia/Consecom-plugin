//go:build !windows

package main

import "errors"

func launchBrowser(exe, dir string, port int) (int, error) {
	return 0, errors.New("Automacao do navegador disponivel somente no Windows")
}
