package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// The inbox is NOT a ChatGPT API. Local built-in actions run every 10 seconds;
// free-form instructions wait for a ChatGPT MCP client or a future opt-in AI worker.
type TaskItem struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Prompt    string    `json:"prompt"`
	Status    string    `json:"status"`
	Result    string    `json:"result,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type TaskInbox struct {
	mu    sync.Mutex
	File  string
	Items []TaskItem
}

func newTaskInbox(dir string) *TaskInbox {
	x := &TaskInbox{File: filepath.Join(dir, "task-inbox.json"), Items: []TaskItem{}}
	if b, e := os.ReadFile(x.File); e == nil {
		_ = json.Unmarshal(b, &x.Items)
	}
	return x
}
func (q *TaskInbox) save() error {
	if e := os.MkdirAll(filepath.Dir(q.File), 0700); e != nil {
		return e
	}
	b, e := json.MarshalIndent(q.Items, "", "  ")
	if e != nil {
		return e
	}
	tmp := q.File + ".tmp"
	if e = os.WriteFile(tmp, b, 0600); e != nil {
		return e
	}
	if e = os.Rename(tmp, q.File); e != nil {
		os.Remove(tmp)
		return e
	}
	return nil
}
func (q *TaskInbox) add(typ, prompt string) (TaskItem, error) {
	prompt = strings.TrimSpace(prompt)
	if len(prompt) < 3 || len(prompt) > 1500 {
		return TaskItem{}, errors.New("pedido deve ter de 3 a 1500 caracteres")
	}
	if typ != "livre" && typ != "diagnostico" && typ != "versao" {
		return TaskItem{}, errors.New("tipo de tarefa invalido")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	now := time.Now()
	t := TaskItem{ID: newID(), Type: typ, Prompt: prompt, Status: "pendente", CreatedAt: now, UpdatedAt: now}
	q.Items = append(q.Items, t)
	if len(q.Items) > 150 {
		q.Items = q.Items[len(q.Items)-150:]
	}
	return t, q.save()
}
func (q *TaskInbox) list() []TaskItem {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]TaskItem{}, q.Items...)
}
func (q *TaskInbox) finish(id, result, status string) (TaskItem, error) {
	if status != "concluida" && status != "erro" {
		return TaskItem{}, errors.New("estado invalido")
	}
	if len(result) > 12000 {
		return TaskItem{}, errors.New("resultado muito extenso")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	for i := range q.Items {
		if q.Items[i].ID == id {
			if q.Items[i].Status != "pendente" {
				return TaskItem{}, errors.New("tarefa nao esta pendente")
			}
			q.Items[i].Status = status
			q.Items[i].Result = result
			q.Items[i].UpdatedAt = time.Now()
			return q.Items[i], q.save()
		}
	}
	return TaskItem{}, errors.New("tarefa nao encontrada")
}
func (q *TaskInbox) runBuiltin(a *App) {
	for _, t := range q.list() {
		if t.Status != "pendente" {
			continue
		}
		if t.Type == "versao" {
			_, _ = q.finish(t.ID, "Consecom Plugin versão "+consecomVersion, "concluida")
		}
		if t.Type == "diagnostico" {
			m, e := systemMetrics()
			if e != nil {
				_, _ = q.finish(t.ID, e.Error(), "erro")
			} else {
				b, _ := json.Marshal(m)
				_, _ = q.finish(t.ID, string(b), "concluida")
			}
		}
	}
}
func (q *TaskInbox) runLoop(a *App, stop <-chan struct{}) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	q.runBuiltin(a)
	for {
		select {
		case <-ticker.C:
			q.runBuiltin(a)
		case <-stop:
			return
		}
	}
}
