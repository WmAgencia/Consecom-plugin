package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

var forbiddenParts = map[string]bool{".git": true, ".ssh": true, ".aws": true, ".azure": true, ".gnupg": true, ".npm": true, "node_modules": true, ".next": true, ".venv": true, "venv": true, "__pycache__": true, "appdata": true, "windows": true, "system32": true, "secrets": true, "credentials": true}
var permittedExt = map[string]bool{".txt": true, ".md": true, ".json": true, ".yaml": true, ".yml": true, ".csv": true, ".html": true, ".css": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true, ".py": true, ".go": true, ".rs": true, ".java": true, ".xml": true, ".log": true, ".toml": true, ".sql": true, ".ps1": true, ".sh": true, ".ini": true, ".vue": true, ".svelte": true, ".php": true, ".rb": true, ".cs": true}

func blockedPart(part string) bool {
	part = strings.ToLower(part)
	if forbiddenParts[part] {
		return true
	}
	if part == ".env" || strings.HasPrefix(part, ".env.") || strings.HasPrefix(part, "id_rsa") || strings.HasPrefix(part, "id_ed25519") || strings.Contains(part, "private_key") || strings.Contains(part, "credential") || strings.Contains(part, "secret") || part == ".npmrc" || part == ".pypirc" || part == ".netrc" || part == "authorized_keys" || part == "known_hosts" || strings.HasSuffix(part, ".pem") || strings.HasSuffix(part, ".key") || strings.HasSuffix(part, ".pfx") || strings.HasSuffix(part, ".p12") {
		return true
	}
	return false
}
func ensureNoLinks(path string) error {
	clean := filepath.Clean(path)
	vol := filepath.VolumeName(clean)
	remaining := strings.TrimPrefix(clean, vol)
	current := vol + string(os.PathSeparator)
	for _, part := range strings.Split(strings.TrimLeft(remaining, `\/`), string(os.PathSeparator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		fi, e := os.Lstat(current)
		if errors.Is(e, os.ErrNotExist) {
			break
		}
		if e != nil {
			return e
		}
		if fi.Mode()&os.ModeSymlink != 0 || fi.Mode()&os.ModeIrregular != 0 {
			return fmt.Errorf("link, junction ou arquivo especial nao permitido: %s", part)
		}
	}
	return nil
}
func safePath(root Root, relative string, forWrite bool) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || filepath.VolumeName(relative) != "" || strings.HasPrefix(relative, `\\`) {
		return "", errors.New("use caminho relativo a pasta autorizada")
	}
	rel := filepath.Clean(relative)
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("caminho fora da pasta")
	}
	for _, part := range strings.FieldsFunc(rel, func(r rune) bool { return r == '/' || r == '\\' }) {
		if blockedPart(part) {
			return "", fmt.Errorf("caminho confidencial bloqueado: %s", part)
		}
	}
	target := filepath.Join(root.Path, rel)
	check, e := filepath.Rel(root.Path, target)
	if e != nil || check == ".." || strings.HasPrefix(check, ".."+string(os.PathSeparator)) {
		return "", errors.New("caminho fora da pasta")
	}
	if e = ensureNoLinks(root.Path); e != nil {
		return "", e
	}
	if e = ensureNoLinks(target); e != nil {
		return "", e
	}
	if forWrite {
		if !root.Writable {
			return "", errors.New("escrita desativada nesta pasta; habilite no painel local")
		}
		if !permittedExt[strings.ToLower(filepath.Ext(target))] {
			return "", errors.New("extensao nao permitida para escrita")
		}
	}
	return target, nil
}
func safeRead(path string) (string, error) {
	fi, e := os.Stat(path)
	if e != nil {
		return "", e
	}
	if !fi.Mode().IsRegular() {
		return "", errors.New("nao e arquivo normal")
	}
	if fi.Size() > 65536 {
		return "", errors.New("arquivo maior que 64 KiB")
	}
	if !permittedExt[strings.ToLower(filepath.Ext(path))] {
		return "", errors.New("tipo de arquivo nao permitido")
	}
	b, e := os.ReadFile(path)
	if e != nil {
		return "", e
	}
	if !utf8.Valid(b) || strings.ContainsRune(string(b), 0) {
		return "", errors.New("somente arquivos de texto UTF-8")
	}
	return string(b), nil
}
func safeList(path string) ([]map[string]any, error) {
	fi, e := os.Stat(path)
	if e != nil {
		return nil, e
	}
	if !fi.IsDir() {
		return nil, errors.New("nao e diretorio")
	}
	list, e := os.ReadDir(path)
	if e != nil {
		return nil, e
	}
	ans := []map[string]any{}
	for _, item := range list {
		if len(ans) >= 120 {
			break
		}
		if blockedPart(item.Name()) || item.Type()&os.ModeSymlink != 0 {
			continue
		}
		if !item.IsDir() && !permittedExt[strings.ToLower(filepath.Ext(item.Name()))] {
			continue
		}
		info, e := item.Info()
		if e != nil || !info.Mode().IsRegular() && !info.IsDir() {
			continue
		}
		ans = append(ans, map[string]any{"name": item.Name(), "dir": item.IsDir(), "size_bytes": info.Size()})
	}
	return ans, nil
}
