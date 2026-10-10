package main

// Recovery vault for destructive file operations. Files stay on the user's PC.
// No remote shell, recursive deletion, or permanent removal is exposed by MCP.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxVaultFile = 16 << 20   // 16 MiB per version
const maxVaultBytes = 256 << 20 // 256 MiB total retained
const maxVaultEntries = 400

type RecoveryEntry struct {
	ID       string    `json:"id"`
	Type     string    `json:"type"` // backup or trash
	Original string    `json:"original"`
	Bytes    int64     `json:"bytes"`
	At       time.Time `json:"created_at"`
	Restored bool      `json:"restored"`
}

type RecoveryResult struct {
	Status     string `json:"status"`
	Path       string `json:"path,omitempty"`
	RecoveryID string `json:"recovery_id,omitempty"`
	Message    string `json:"message,omitempty"`
}

func (a *App) vaultDir() string           { return filepath.Join(filepath.Dir(a.path), "recovery") }
func (a *App) vaultManifest() string      { return filepath.Join(a.vaultDir(), "index.json") }
func (a *App) vaultBlob(id string) string { return filepath.Join(a.vaultDir(), id+".blob") }

func (a *App) loadVaultLocked() ([]RecoveryEntry, error) {
	b, e := os.ReadFile(a.vaultManifest())
	if errors.Is(e, os.ErrNotExist) {
		return []RecoveryEntry{}, nil
	}
	if e != nil {
		return nil, e
	}
	var list []RecoveryEntry
	if e = json.Unmarshal(b, &list); e != nil {
		return nil, fmt.Errorf("indice de backups danificado: %w", e)
	}
	return list, nil
}

func (a *App) saveVaultLocked(list []RecoveryEntry) error {
	if e := os.MkdirAll(a.vaultDir(), 0700); e != nil {
		return e
	}
	b, e := json.MarshalIndent(list, "", "  ")
	if e != nil {
		return e
	}
	tmp := a.vaultManifest() + ".tmp-" + newID()
	if e = os.WriteFile(tmp, b, 0600); e != nil {
		return e
	}
	if e = os.Rename(tmp, a.vaultManifest()); e != nil {
		_ = os.Remove(tmp)
		return e
	}
	return nil
}

func regularFile(path string) (os.FileInfo, error) {
	if e := ensureNoLinks(path); e != nil {
		return nil, e
	}
	info, e := os.Lstat(path)
	if e != nil {
		return nil, e
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("apenas arquivo comum e permitido nesta operacao")
	}
	if info.Size() > maxVaultFile {
		return nil, errors.New("arquivo acima do limite recuperavel de 16 MiB")
	}
	return info, nil
}

// Lock is held across operations to keep the on-disk manifest consistent.
func (a *App) archiveLocked(path, kind string) (RecoveryEntry, error) {
	fi, e := regularFile(path)
	if e != nil {
		return RecoveryEntry{}, e
	}
	list, e := a.loadVaultLocked()
	if e != nil {
		return RecoveryEntry{}, e
	}
	var used int64
	for _, x := range list {
		used += x.Bytes
	}
	if len(list) >= maxVaultEntries || used+fi.Size() > maxVaultBytes {
		return RecoveryEntry{}, errors.New("cofre cheio (400 versoes ou 256 MiB); libere espaco manualmente no Windows antes de continuar")
	}
	b, e := os.ReadFile(path)
	if e != nil {
		return RecoveryEntry{}, e
	}
	id := newID()
	if e = os.MkdirAll(a.vaultDir(), 0700); e != nil {
		return RecoveryEntry{}, e
	}
	blob := a.vaultBlob(id)
	if e = os.WriteFile(blob, b, 0600); e != nil {
		return RecoveryEntry{}, e
	}
	item := RecoveryEntry{ID: id, Type: kind, Original: path, Bytes: int64(len(b)), At: time.Now()}
	list = append(list, item)
	if e = a.saveVaultLocked(list); e != nil {
		_ = os.Remove(blob)
		return RecoveryEntry{}, e
	}
	return item, nil
}
func (a *App) archiveFile(path, kind string) (RecoveryEntry, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.archiveLocked(path, kind)
}
func (a *App) recoveryList() ([]RecoveryEntry, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	list, e := a.loadVaultLocked()
	if e != nil {
		return nil, e
	}
	sort.Slice(list, func(i, j int) bool { return list[i].At.After(list[j].At) })
	return list, nil
}

func parentExists(target string) error {
	fi, e := os.Lstat(filepath.Dir(target))
	if e != nil {
		return e
	}
	if !fi.IsDir() {
		return errors.New("pasta pai invalida")
	}
	return ensureNoLinks(filepath.Dir(target))
}

func (a *App) createFolder(path string) (RecoveryResult, error) {
	target, e := safeFullPath(path, false)
	if e != nil {
		return RecoveryResult{}, e
	}
	if filepath.Ext(target) == "." || strings.HasPrefix(filepath.Base(target), ".") {
		return RecoveryResult{}, errors.New("nome de pasta oculto/protegido bloqueado")
	}
	if e = parentExists(target); e != nil {
		return RecoveryResult{}, e
	}
	if _, e = os.Lstat(target); e == nil {
		return RecoveryResult{}, errors.New("pasta ou arquivo ja existe")
	}
	if !errors.Is(e, os.ErrNotExist) {
		return RecoveryResult{}, e
	}
	if e = os.Mkdir(target, 0700); e != nil {
		return RecoveryResult{}, e
	}
	return RecoveryResult{Status: "pasta_criada", Path: target}, nil
}

func (a *App) moveFile(from, to string) (RecoveryResult, error) {
	src, e := safeFullPath(from, true)
	if e != nil {
		return RecoveryResult{}, e
	}
	dst, e := safeFullPath(to, true)
	if e != nil {
		return RecoveryResult{}, e
	}
	if strings.EqualFold(src, dst) {
		return RecoveryResult{}, errors.New("origem e destino iguais")
	}
	if _, e = regularFile(src); e != nil {
		return RecoveryResult{}, e
	}
	if e = parentExists(dst); e != nil {
		return RecoveryResult{}, e
	}
	if _, e = os.Lstat(dst); e == nil {
		return RecoveryResult{}, errors.New("destino ja existe; nao sobreescrevemos arquivos ao mover")
	}
	if !errors.Is(e, os.ErrNotExist) {
		return RecoveryResult{}, e
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	backup, e := a.archiveLocked(src, "backup")
	if e != nil {
		return RecoveryResult{}, e
	}
	if e = os.Rename(src, dst); e == nil {
		return RecoveryResult{Status: "arquivo_movido", Path: dst, RecoveryID: backup.ID}, nil
	}
	// Cross-volume fallback: verify the source, create destination exclusively, and remove source only if copy succeeds.
	data, e := os.ReadFile(src)
	if e != nil {
		return RecoveryResult{}, e
	}
	out, e := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return RecoveryResult{}, e
	}
	n, writeErr := out.Write(data)
	closeErr := out.Close()
	if writeErr != nil || closeErr != nil || n != len(data) {
		_ = os.Remove(dst)
		return RecoveryResult{}, errors.New("falha ao copiar arquivo entre discos; origem preservada")
	}
	if e = os.Remove(src); e != nil {
		_ = os.Remove(dst)
		return RecoveryResult{}, fmt.Errorf("origem preservada; falha no movimento: %w", e)
	}
	return RecoveryResult{Status: "arquivo_movido", Path: dst, RecoveryID: backup.ID}, nil
}

func (a *App) trashFile(path string) (RecoveryResult, error) {
	target, e := safeFullPath(path, true)
	if e != nil {
		return RecoveryResult{}, e
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	entry, e := a.archiveLocked(target, "trash")
	if e != nil {
		return RecoveryResult{}, e
	}
	if e = os.Remove(target); e != nil {
		return RecoveryResult{}, fmt.Errorf("arquivo original preservado; falha ao mover para lixeira: %w", e)
	}
	return RecoveryResult{Status: "enviado_lixeira_recuperavel", Path: target, RecoveryID: entry.ID}, nil
}
func (a *App) removeEmptyFolder(path string) (RecoveryResult, error) {
	target, e := safeFullPath(path, false)
	if e != nil {
		return RecoveryResult{}, e
	}
	if e = ensureNoLinks(target); e != nil {
		return RecoveryResult{}, e
	}
	fi, e := os.Lstat(target)
	if e != nil {
		return RecoveryResult{}, e
	}
	if !fi.IsDir() {
		return RecoveryResult{}, errors.New("nao e pasta")
	}
	children, e := os.ReadDir(target)
	if e != nil {
		return RecoveryResult{}, e
	}
	if len(children) != 0 {
		return RecoveryResult{}, errors.New("pasta nao esta vazia; exclusao recursiva e bloqueada")
	}
	if e = os.Remove(target); e != nil {
		return RecoveryResult{}, e
	}
	return RecoveryResult{Status: "pasta_vazia_removida", Path: target}, nil
}

func (a *App) restoreRecovery(id string) (RecoveryResult, error) {
	if len(id) != 32 || strings.Trim(id, "0123456789abcdef") != "" {
		return RecoveryResult{}, errors.New("ID de recuperacao invalido")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	list, e := a.loadVaultLocked()
	if e != nil {
		return RecoveryResult{}, e
	}
	idx := -1
	for i, item := range list {
		if item.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return RecoveryResult{}, errors.New("backup nao encontrado")
	}
	item := list[idx]
	if item.Restored {
		return RecoveryResult{}, errors.New("esta versao ja foi restaurada")
	}
	target, e := safeFullPath(item.Original, true)
	if e != nil {
		return RecoveryResult{}, e
	}
	if e = parentExists(target); e != nil {
		return RecoveryResult{}, e
	}
	blob, e := os.ReadFile(a.vaultBlob(id))
	if e != nil {
		return RecoveryResult{}, e
	}
	if int64(len(blob)) != item.Bytes {
		return RecoveryResult{}, errors.New("backup corrompido; nada foi alterado")
	}
	_, statErr := os.Lstat(target)
	if statErr == nil {
		if item.Type == "trash" {
			return RecoveryResult{}, errors.New("destino ja existe; lixeira nao sobrescreve o arquivo atual")
		}
		// Back up the version we're about to replace, for safe undo of restore.
		if _, e = a.archiveLocked(target, "backup"); e != nil {
			return RecoveryResult{}, e
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return RecoveryResult{}, statErr
	}
	tmp := target + ".mcp-restore-" + newID()
	if e = os.WriteFile(tmp, blob, 0600); e != nil {
		return RecoveryResult{}, e
	}
	if e = os.Rename(tmp, target); e != nil {
		_ = os.Remove(tmp)
		return RecoveryResult{}, e
	}
	list, e = a.loadVaultLocked()
	if e != nil {
		return RecoveryResult{}, e
	}
	for i := range list {
		if list[i].ID == id {
			list[i].Restored = true
			break
		}
	}
	if e = a.saveVaultLocked(list); e != nil {
		return RecoveryResult{}, fmt.Errorf("arquivo restaurado mas nao foi possivel atualizar o indice: %w", e)
	}
	return RecoveryResult{Status: "restaurado", Path: target, RecoveryID: id}, nil
}

func (a *App) fileOperation(name string, args map[string]json.RawMessage) (any, error) {
	if name == "listar_recuperacao_pc" {
		return a.recoveryList()
	}
	if a.mode() != "maximum" {
		return nil, errors.New("operacao exige aprovacao maxima habilitada no painel local")
	}
	switch name {
	case "criar_pasta_pc", "excluir_pasta_vazia_pc", "excluir_arquivo_pc", "criar_backup_pc":
		path, e := argStr(args, "caminho_absoluto")
		if e != nil {
			return nil, e
		}
		switch name {
		case "criar_pasta_pc":
			return a.createFolder(path)
		case "excluir_pasta_vazia_pc":
			return a.removeEmptyFolder(path)
		case "excluir_arquivo_pc":
			return a.trashFile(path)
		default:
			target, e := safeFullPath(path, true)
			if e != nil {
				return nil, e
			}
			entry, e := a.archiveFile(target, "backup")
			if e != nil {
				return nil, e
			}
			return RecoveryResult{Status: "backup_criado", Path: target, RecoveryID: entry.ID}, nil
		}
	case "mover_arquivo_pc":
		from, e := argStr(args, "origem")
		if e != nil {
			return nil, e
		}
		to, e := argStr(args, "destino")
		if e != nil {
			return nil, e
		}
		return a.moveFile(from, to)
	case "restaurar_recuperacao_pc":
		id, e := argStr(args, "id")
		if e != nil {
			return nil, e
		}
		return a.restoreRecovery(id)
	}
	return nil, errors.New("operacao desconhecida")
}
