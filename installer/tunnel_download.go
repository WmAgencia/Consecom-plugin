//go:build windows

package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const tunnelReleaseBase = "https://github.com/openai/tunnel-client/releases/download/v0.0.16/"
const tunnelZipName = "tunnel-client-v0.0.16-windows-amd64.zip"

// The archive and official SHA256SUMS file are downloaded over HTTPS directly
// from the OpenAI GitHub release. Only tunnel-client.exe is installed.
func downloadOfficialTunnelClient(dir string) error {
	target := filepath.Join(dir, "tunnel-client.exe")
	if f, e := os.Stat(target); e == nil && f.Size() > 500000 {
		return nil
	}
	client := http.Client{Timeout: 150 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 7 {
			return errors.New("muitos redirecionamentos")
		}
		if req.URL.Scheme != "https" {
			return errors.New("redirecionamento sem HTTPS bloqueado")
		}
		h := strings.ToLower(req.URL.Hostname())
		if h != "github.com" && !strings.HasSuffix(h, ".githubusercontent.com") && h != "release-assets.githubusercontent.com" {
			return fmt.Errorf("destino inesperado: %s", h)
		}
		return nil
	}}
	retrieve := func(path string, limit int64) ([]byte, error) {
		r, e := client.Get(tunnelReleaseBase + path)
		if e != nil {
			return nil, e
		}
		defer r.Body.Close()
		if r.StatusCode != 200 {
			return nil, fmt.Errorf("download retornou HTTP %d", r.StatusCode)
		}
		b, e := io.ReadAll(io.LimitReader(r.Body, limit+1))
		if e != nil {
			return nil, e
		}
		if int64(len(b)) > limit {
			return nil, errors.New("arquivo excedeu o limite")
		}
		return b, nil
	}
	sums, e := retrieve("SHA256SUMS.txt", 1024*1024)
	if e != nil {
		return fmt.Errorf("checksum oficial: %w", e)
	}
	expected := ""
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 2 && strings.TrimLeft(fields[len(fields)-1], "*") == tunnelZipName {
			expected = fields[0]
			break
		}
	}
	if len(expected) != 64 {
		return errors.New("checksum do cliente Windows nao encontrado na lista oficial")
	}
	zipBytes, e := retrieve(tunnelZipName, 42*1024*1024)
	if e != nil {
		return fmt.Errorf("cliente de tunel: %w", e)
	}
	digest := sha256.Sum256(zipBytes)
	if !strings.EqualFold(hex.EncodeToString(digest[:]), expected) {
		return errors.New("falha de integridade do pacote oficial (SHA-256)")
	}
	z, e := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if e != nil {
		return e
	}
	for _, f := range z.File {
		if !strings.EqualFold(filepath.Base(strings.ReplaceAll(f.Name, "/", string(filepath.Separator))), "tunnel-client.exe") {
			continue
		}
		if f.UncompressedSize64 < 200000 || f.UncompressedSize64 > 90*1024*1024 {
			return errors.New("executavel oficial com tamanho inesperado")
		}
		rc, e := f.Open()
		if e != nil {
			return e
		}
		tmp := target + ".download"
		out, e := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0700)
		if e != nil {
			rc.Close()
			return e
		}
		n, copyErr := io.CopyN(out, rc, 91*1024*1024)
		closeErr := out.Close()
		rc.Close()
		if copyErr != nil && copyErr != io.EOF {
			os.Remove(tmp)
			return copyErr
		}
		if closeErr != nil {
			os.Remove(tmp)
			return closeErr
		}
		if n < 200000 || n > 90*1024*1024 {
			os.Remove(tmp)
			return errors.New("conteudo de executavel invalido")
		}
		if e = os.Rename(tmp, target); e != nil {
			os.Remove(tmp)
			return e
		}
		return nil
	}
	return errors.New("cliente tunnel-client.exe ausente no ZIP oficial")
}
