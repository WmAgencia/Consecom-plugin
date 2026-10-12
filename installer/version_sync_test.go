//go:build windows

package main

import (
 "os"
 "path/filepath"
 "regexp"
 "strings"
 "testing"
)

// Regression: stale health-check 4.14.7 accidentally rolled back the 4.14.8 server.
func TestInstallerVersionMatchesServer(t *testing.T) {
 source, err:=os.ReadFile(filepath.Join("..","update.go"))
 if err!=nil {t.Fatal(err)}
 rx:=regexp.MustCompile(`const consecomVersion\s*=\s*"([^"]+)"`)
 found:=rx.FindStringSubmatch(string(source))
 if len(found)!=2 {t.Fatal("could not find server version in update.go")}
 if installerExpectedVersion!=found[1] {
  t.Fatalf("installer expects %q, server identifies as %q; installer would roll back the correct executable",installerExpectedVersion,found[1])
 }
 code, err:=os.ReadFile("main.go")
 if err!=nil {t.Fatal(err)}
 if !strings.Contains(string(code),"state.Version == installerExpectedVersion") {
  t.Fatal("installer health check must use expected-version constant")
 }
}
