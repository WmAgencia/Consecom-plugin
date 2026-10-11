package main

import (
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
 "path/filepath"
)

func TestEmbeddedConversationAssets(t *testing.T) {
 a,e:=NewApp(filepath.Join(t.TempDir(),"config.json"));if e!=nil {t.Fatal(e)}
 for _,test:=range []struct{path,ctype,fragment string}{
 {"/chat-ui.css","text/css","chat-dock"},
 {"/chat-ui.js","application/javascript","chatHistory"},
 {"/","text/html","/chat-ui.js"},
 }{
 r:=httptest.NewRecorder()
 a.handler().ServeHTTP(r,httptest.NewRequest(http.MethodGet,"http://127.0.0.1:8765"+test.path,nil))
 if r.Code!=http.StatusOK { t.Fatalf("%s devolveu %d: %s",test.path,r.Code,r.Body.String()) }
 if !strings.Contains(r.Header().Get("Content-Type"),test.ctype){t.Fatalf("content-type errado em %s: %s",test.path,r.Header().Get("Content-Type"))}
 if !strings.Contains(r.Body.String(),test.fragment){t.Fatalf("recurso %s nao contem %q",test.path,test.fragment)}
 }
}
