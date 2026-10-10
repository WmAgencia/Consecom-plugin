package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Chrome 144+ can authorize remote debugging of the already-running default
// profile via chrome://inspect/#remote-debugging. Chrome writes a short-lived
// DevToolsActivePort file, NOT a copy of the user's cookies/passwords.
// The Chrome permission UI must be approved locally; the MCP cannot bypass it.
func liveChromeWebSocket() (string, error) {
	root := browserDataRoot("chrome")
	if root == "" {
		return "", errors.New("diretorio do Chrome nao encontrado")
	}
	p := filepath.Join(root, "DevToolsActivePort")
	b, err := os.ReadFile(p)
	if err != nil {
		return "", errors.New("Chrome ja aberto: em seu Chrome (versao 144+), abra chrome://inspect/#remote-debugging e ative a Depuracao Remota. Depois aprove a conexao local no Chrome e tente novamente. Se usar varios perfis, este metodo seleciona apenas a instancia padrao do Chrome")
	}
	parts := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(parts) < 2 {
		return "", errors.New("arquivo DevToolsActivePort invalido; desative e reative Depuracao Remota no Chrome")
	}
	port, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || port < 1 || port > 65535 {
		return "", errors.New("porta de depuracao do Chrome invalida")
	}
	path := strings.TrimSpace(parts[1])
	if !strings.HasPrefix(path, "/devtools/browser/") || len(path) > 300 || strings.ContainsAny(path, "?#\r\n\t") {
		return "", errors.New("endpoint de depuracao do Chrome invalido")
	}
	return fmt.Sprintf("ws://127.0.0.1:%d%s", port, path), nil
}

func createLiveBrowserTarget(ws string) (string, error) {
	response, err := cdpBrowserCommand(ws, "Target.createTarget", map[string]any{"url": "about:blank"}, "")
	if err != nil {
		return "", fmt.Errorf("nao foi possivel conectar ao Chrome ja aberto: %w. Abra chrome://inspect/#remote-debugging e autorize o acesso", err)
	}
	var result struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(response, &result); err != nil || result.TargetID == "" {
		return "", errors.New("Chrome retornou destino de aba invalido")
	}
	return result.TargetID, nil
}

func browserMessage(w *cdpWire, id int, method string, params any, sessionID string) (json.RawMessage, error) {
	req := map[string]any{"id": id, "method": method, "params": params}
	if sessionID != "" {
		req["sessionId"] = sessionID
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if err = w.writeFrame(1, raw); err != nil {
		return nil, err
	}
	for i := 0; i < 350; i++ {
		b, opcode, err := w.readFrame()
		if err != nil {
			return nil, err
		}
		if opcode == 9 {
			_ = w.writeFrame(10, b)
			continue
		}
		if opcode == 8 {
			return nil, errors.New("Chrome encerrou a conexao DevTools")
		}
		if opcode != 1 {
			continue
		}
		var response struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(b, &response) != nil || response.ID != id {
			continue
		}
		if response.Error != nil {
			return nil, errors.New(response.Error.Message)
		}
		return response.Result, nil
	}
	return nil, errors.New("Chrome nao respondeu ao comando DevTools")
}

// For live Chrome, use the browser WebSocket and flatten a short-lived CDP
// session to ONLY the fresh tab that Consecom created. No other tab is read.
func cdpBrowserCommand(ws, method string, params any, targetID string) (json.RawMessage, error) {
	w, err := openCDP(ws)
	if err != nil {
		return nil, fmt.Errorf("Chrome nao autorizado ou depuracao desligada: %w", err)
	}
	defer w.conn.Close()
	_ = w.conn.SetDeadline(time.Now().Add(17 * time.Second))
	sessionID := ""
	if targetID != "" {
		response, err := browserMessage(w, 1, "Target.attachToTarget", map[string]any{"targetId": targetID, "flatten": true}, "")
		if err != nil {
			return nil, err
		}
		var got struct {
			SessionID string `json:"sessionId"`
		}
		if json.Unmarshal(response, &got) != nil || got.SessionID == "" {
			return nil, errors.New("sessao CDP de aba invalida")
		}
		sessionID = got.SessionID
	}
	id := 1
	if sessionID != "" {
		id = 2
	}
	return browserMessage(w, id, method, params, sessionID)
}
