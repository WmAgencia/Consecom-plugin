package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"
)

func ioReadLimit(r io.Reader, max int64) ([]byte, error) { return io.ReadAll(io.LimitReader(r, max+1)) }

type cdpWire struct {
	conn   net.Conn
	reader *bufio.Reader
}

func openCDP(address string) (*cdpWire, error) {
	u, e := url.Parse(address)
	if e != nil || u.Scheme != "ws" || u.Hostname() != "127.0.0.1" {
		return nil, errors.New("websocket CDP deve ser local")
	}
	conn, e := net.DialTimeout("tcp", u.Host, 3*time.Second)
	if e != nil {
		return nil, e
	}
	fail := func(err error) (*cdpWire, error) { conn.Close(); return nil, err }
	var nonce [16]byte
	rand.Read(nonce[:])
	key := base64.StdEncoding.EncodeToString(nonce[:])
	path := u.RequestURI()
	_, e = fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: %s\r\n\r\n", path, u.Host, key)
	if e != nil {
		return fail(e)
	}
	_ = conn.SetDeadline(time.Now().Add(6 * time.Second))
	r := bufio.NewReader(conn)
	response, e := ioReadHTTPResponse(r)
	if e != nil {
		return fail(e)
	}
	h := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	expected := base64.StdEncoding.EncodeToString(h[:])
	if !strings.HasPrefix(response, "HTTP/1.1 101 ") || !strings.Contains(strings.ToLower(response), "sec-websocket-accept: "+strings.ToLower(expected)) {
		return fail(fmt.Errorf("handshake websocket recusado: %q", response))
	}
	return &cdpWire{conn: conn, reader: r}, nil
}
func ioReadHTTPResponse(r *bufio.Reader) (string, error) {
	out := ""
	for i := 0; i < 35; i++ {
		s, e := r.ReadString('\n')
		if e != nil {
			return "", e
		}
		out += s
		if s == "\r\n" {
			return out, nil
		}
	}
	return "", errors.New("headers websocket muito longos")
}
func (w *cdpWire) writeFrame(opcode byte, data []byte) error {
	if len(data) > 8*1024*1024 {
		return errors.New("frame muito grande")
	}
	head := []byte{0x80 | opcode}
	switch {
	case len(data) < 126:
		head = append(head, 0x80|byte(len(data)))
	case len(data) <= 65535:
		head = append(head, 0xfe, byte(len(data)>>8), byte(len(data)))
	default:
		head = append(head, 0xff)
		var n [8]byte
		binary.BigEndian.PutUint64(n[:], uint64(len(data)))
		head = append(head, n[:]...)
	}
	var mask [4]byte
	rand.Read(mask[:])
	head = append(head, mask[:]...)
	payload := make([]byte, len(data))
	for i, b := range data {
		payload[i] = b ^ mask[i%4]
	}
	head = append(head, payload...)
	_, e := w.conn.Write(head)
	return e
}
func (w *cdpWire) readFrame() ([]byte, byte, error) {
	var hdr [2]byte
	if _, e := io.ReadFull(w.reader, hdr[:]); e != nil {
		return nil, 0, e
	}
	if hdr[0]&0x80 == 0 {
		return nil, 0, errors.New("CDP fragmentado inesperado")
	}
	opcode := hdr[0] & 0x0f
	n := uint64(hdr[1] & 0x7f)
	if n == 126 {
		var v [2]byte
		if _, e := io.ReadFull(w.reader, v[:]); e != nil {
			return nil, 0, e
		}
		n = uint64(binary.BigEndian.Uint16(v[:]))
	} else if n == 127 {
		var v [8]byte
		if _, e := io.ReadFull(w.reader, v[:]); e != nil {
			return nil, 0, e
		}
		n = binary.BigEndian.Uint64(v[:])
	}
	if n > 8*1024*1024 {
		return nil, 0, errors.New("resposta CDP acima do limite")
	}
	var mask [4]byte
	if hdr[1]&0x80 != 0 {
		if _, e := io.ReadFull(w.reader, mask[:]); e != nil {
			return nil, 0, e
		}
	}
	data := make([]byte, int(n))
	if _, e := io.ReadFull(w.reader, data); e != nil {
		return nil, 0, e
	}
	if hdr[1]&0x80 != 0 {
		for i := range data {
			data[i] ^= mask[i%4]
		}
	}
	return data, opcode, nil
}
func cdpCall(s *BrowserSession, method string, params any) (json.RawMessage, error) {
	if s.BrowserWS != "" {
		return cdpBrowserCommand(s.BrowserWS, method, params, s.Target)
	}
	ws, e := browserCurrent(s)
	if e != nil {
		return nil, e
	}
	w, e := openCDP(ws)
	if e != nil {
		return nil, e
	}
	defer w.conn.Close()
	_ = w.conn.SetDeadline(time.Now().Add(13 * time.Second))
	req := map[string]any{"id": 1, "method": method, "params": params}
	p, e := json.Marshal(req)
	if e != nil {
		return nil, e
	}
	if e = w.writeFrame(1, p); e != nil {
		return nil, e
	}
	for i := 0; i < 300; i++ {
		b, opcode, er := w.readFrame()
		if er != nil {
			return nil, er
		}
		if opcode == 9 {
			_ = w.writeFrame(10, b)
			continue
		}
		if opcode == 8 {
			return nil, errors.New("CDP fechou a conexao")
		}
		if opcode != 1 {
			continue
		}
		var res struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(b, &res) != nil || res.ID != 1 {
			continue
		}
		if res.Error != nil {
			return nil, errors.New(res.Error.Message)
		}
		return res.Result, nil
	}
	return nil, errors.New("CDP: sem resposta")
}
func cdpEvaluate(s *BrowserSession, expression string) (any, error) {
	res, e := cdpCall(s, "Runtime.evaluate", map[string]any{"expression": expression, "returnByValue": true, "awaitPromise": true, "userGesture": true})
	if e != nil {
		return nil, e
	}
	var v struct {
		Result struct {
			Type        string          `json:"type"`
			Value       json.RawMessage `json:"value"`
			Description string          `json:"description"`
		} `json:"result"`
		ExceptionDetails json.RawMessage `json:"exceptionDetails"`
	}
	if json.Unmarshal(res, &v) != nil {
		return nil, errors.New("resposta Runtime.evaluate invalida")
	}
	if len(v.ExceptionDetails) > 0 {
		return nil, fmt.Errorf("JavaScript retornou uma excecao: %s", string(v.ExceptionDetails)[:min(400, len(v.ExceptionDetails))])
	}
	var out any
	if len(v.Result.Value) > 0 {
		if e = json.Unmarshal(v.Result.Value, &out); e != nil {
			return nil, e
		}
	}
	return out, nil
}
func jsonJS(s string) string { b, _ := json.Marshal(s); return string(b) }
func browserNavigateLocked(s *BrowserSession, raw string) (any, error) {
	_, e := cdpCall(s, "Page.navigate", map[string]any{"url": raw})
	if e != nil {
		return nil, e
	}
	time.Sleep(380 * time.Millisecond)
	for i := 0; i < 10; i++ {
		v, e := cdpEvaluate(s, "document.readyState")
		if e == nil && (v == "interactive" || v == "complete") {
			break
		}
		time.Sleep(220 * time.Millisecond)
	}
	snap, e := browserSnapshotLocked(s)
	if e != nil {
		return map[string]any{"status": "navegacao_iniciada", "url": raw, "warning": e.Error()}, nil
	}
	return snap, nil
}
func browserSnapshotLocked(s *BrowserSession) (any, error) {
	return cdpEvaluate(s, `(()=>({url:location.href,title:document.title,text:(document.body?.innerText||'').slice(0,18000),elements:Array.from(document.querySelectorAll('a,button,input,select,textarea,[role="button"]')).slice(0,80).map((el,i)=>({tag:el.tagName.toLowerCase(),id:el.id||'',name:el.getAttribute('name')||'',type:el.getAttribute('type')||'',text:(el.innerText||el.getAttribute('aria-label')||el.getAttribute('placeholder')||'').slice(0,100)}))}))()`)
}
func browserClickLocked(s *BrowserSession, sel string) (any, error) {
	if len(sel) == 0 || len(sel) > 350 {
		return nil, errors.New("seletor CSS invalido")
	}
	code := `(()=>{const el=document.querySelector(` + jsonJS(sel) + `);if(!el)return {ok:false,reason:'elemento nao encontrado'};let tag=(el.tagName||'').toLowerCase(),type=(el.getAttribute('type')||'').toLowerCase(),t=(el.innerText||el.getAttribute('aria-label')||'').toLowerCase();if(type==='password')return {ok:false,reason:'campo de senha protegido'};if(el.matches('button[type="submit"],input[type="submit"],button[formaction]')||/comprar|pagar|excluir|deletar|enviar|publicar|finalizar compra|subscribe|purchase|checkout|delete|send|publish/.test(t))return {ok:false,reason:'acao sensivel/submit: confirmar com usuario e usar fluxo de aprovacao local (nao implementado nesta versao)'};el.click();return {ok:true,clicked:tag,text:t.slice(0,100),url:location.href}})()`
	return cdpEvaluate(s, code)
}
func browserTypeLocked(s *BrowserSession, sel, text string) (any, error) {
	if len(sel) == 0 || len(sel) > 350 || len(text) > 4096 {
		return nil, errors.New("seletor/texto invalido ou acima do limite")
	}
	code := `(()=>{const el=document.querySelector(` + jsonJS(sel) + `);if(!el)return {ok:false,reason:'elemento nao encontrado'};if(el.matches('input[type="password"],[autocomplete="cc-number"],[autocomplete="cc-csc"]'))return {ok:false,reason:'dados sensiveis bloqueados'};if(!el.matches('input,textarea,[contenteditable="true"]'))return {ok:false,reason:'elemento nao editavel'};let value=` + jsonJS(text) + `;if(el.isContentEditable){el.textContent=value;el.dispatchEvent(new InputEvent('input',{bubbles:true,data:value,inputType:'insertText'}))}else{const descriptor=Object.getOwnPropertyDescriptor(el.tagName==='TEXTAREA'?HTMLTextAreaElement.prototype:HTMLInputElement.prototype,'value');descriptor.set.call(el,value);el.dispatchEvent(new Event('input',{bubbles:true}));el.dispatchEvent(new Event('change',{bubbles:true}))}return {ok:true,chars:value.length}})()`
	return cdpEvaluate(s, code)
}
func browserScrollLocked(s *BrowserSession, pixels int) (any, error) {
	return cdpEvaluate(s, fmt.Sprintf(`(()=>{window.scrollBy(0,%d);return {ok:true,y:Math.round(window.scrollY)}})()`, pixels))
}
func browserScreenshotLocked(s *BrowserSession) (any, error) {
	res, e := cdpCall(s, "Page.captureScreenshot", map[string]any{"format": "jpeg", "quality": 68, "captureBeyondViewport": false})
	if e != nil {
		return nil, e
	}
	var v struct {
		Data string `json:"data"`
	}
	if e = json.Unmarshal(res, &v); e != nil {
		return nil, e
	}
	if len(v.Data) > 1800000 {
		return nil, errors.New("captura de tela muito grande")
	}
	if _, e = base64.StdEncoding.DecodeString(v.Data); e != nil {
		return nil, e
	}
	return BrowserScreenshot{Data: v.Data}, nil
}
