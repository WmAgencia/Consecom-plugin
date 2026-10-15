package main

import (
    "encoding/json"
    "path/filepath"
    "strings"
    "testing"
)

func TestDesktopRelayExportsEntireMCPToolSet(t *testing.T) {
    catalog := tools()
    if len(catalog) < 110 {
        t.Fatalf("remote registry shrank: found %d local tools, expected at least 110", len(catalog))
    }
    seen := map[string]bool{}
    for _, spec := range catalog {
        if spec.Name == "" || seen[spec.Name] {
            t.Fatalf("invalid/duplicate tool in registry: %s", spec.Name)
        }
        seen[spec.Name] = true
        if spec.InputSchema == nil || spec.Description == "" || spec.Annotations == nil {
            t.Fatalf("incomplete MCP tool definition: %s", spec.Name)
        }
    }
    for _, name := range []string{"diagnostico_pc", "excluir_arquivo_pc", "browser_abrir", "desktop_clicar", "video_render_iniciar", "fluxo_iniciar"} {
        if !seen[name] { t.Fatalf("missing original tool: %s", name) }
    }
}

func TestDesktopRelayDispatchForwardsToLocalGuards(t *testing.T) {
    a, err := NewApp(filepath.Join(t.TempDir(), "config.json"))
    if err != nil { t.Fatal(err) }
    if _,err = desktopExecuteTool(a, desktopJob{Tool:"nao_existe",Arguments:json.RawMessage("{}")});err==nil {
        t.Fatal("remote job accepted unregistered tool")
    }
    if _,err = desktopExecuteTool(a, desktopJob{Tool:"diagnostico_pc", Arguments:json.RawMessage("{}")});err!=nil {
        t.Fatalf("read tool dispatch failed: %v",err)
    }
    // A destructive operation with missing arguments must be rejected by the
    // underlying local dispatcher, NOT by the old read-only transport whitelist.
    _,err=desktopExecuteTool(a,desktopJob{Tool:"excluir_arquivo_pc",Arguments:json.RawMessage("{}")})
    if err!=nil && strings.Contains(err.Error(),"ferramenta nao registrada") {
        t.Fatalf("destructive tool still blocked by transport: %v",err)
    }
    if _,err=desktopExecuteTool(a,desktopJob{Tool:"diagnostico_pc",Arguments:json.RawMessage(strings.Repeat("x",65001))});err==nil {
        t.Fatal("oversized remote job accepted")
    }
}
