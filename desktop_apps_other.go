//go:build !windows

package main

import "errors"

type InstalledWindowsApp struct {
 Name string `json:"name"`
 AppID string `json:"app_id"`
}

var errWindowsAppsOnly=errors.New("abertura de aplicativos disponivel apenas no Windows")

func windowsAppsSearch(_ string)(any,error){return nil,errWindowsAppsOnly}
func validRegisteredApp(_ string)(InstalledWindowsApp,error){return InstalledWindowsApp{},errWindowsAppsOnly}
func launchRegisteredWindowsApp(_ string)(string,error){return "",errWindowsAppsOnly}
func validatePortableWindowsExe(_ string)(string,error){return "",errWindowsAppsOnly}
func launchPortableWindowsExe(_ string)(string,error){return "",errWindowsAppsOnly}
