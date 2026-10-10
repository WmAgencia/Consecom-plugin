//go:build windows

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var globalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
var getSystemTimes = kernel32.NewProc("GetSystemTimes")
var processMemoryInfo = syscall.NewLazyDLL("psapi.dll").NewProc("GetProcessMemoryInfo")

const createNoWindow = 0x08000000

func hideConsole(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}

// WORKING_SET em bytes retornado por GetProcessMemoryInfo (somente leitura).
type processMemoryCounters struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uint64
	WorkingSetSize             uint64
	QuotaPeakPagedPoolUsage    uint64
	QuotaPagedPoolUsage        uint64
	QuotaPeakNonPagedPoolUsage uint64
	QuotaNonPagedPoolUsage     uint64
	PagefileUsage              uint64
	PeakPagefileUsage          uint64
}

func workingSet(pid uint32) (uint64, error) {
	const access = 0x0400 | 0x0010 // PROCESS_QUERY_INFORMATION + PROCESS_VM_READ
	h, e := syscall.OpenProcess(access, false, pid)
	if e != nil {
		return 0, e
	}
	defer syscall.CloseHandle(h)
	m := processMemoryCounters{CB: uint32(unsafe.Sizeof(processMemoryCounters{}))}
	ok, _, err := processMemoryInfo.Call(uintptr(h), uintptr(unsafe.Pointer(&m)), uintptr(m.CB))
	if ok == 0 {
		return 0, err
	}
	return m.WorkingSetSize, nil
}

func runningProcesses() ([]psProcess, error) {
	h, e := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if e != nil {
		return nil, e
	}
	defer syscall.CloseHandle(h)
	entry := syscall.ProcessEntry32{Size: uint32(unsafe.Sizeof(syscall.ProcessEntry32{}))}
	if e := syscall.Process32First(h, &entry); e != nil {
		return nil, e
	}
	list := make([]psProcess, 0, 128)
	for {
		name := syscall.UTF16ToString(entry.ExeFile[:])
		if strings.HasSuffix(strings.ToLower(name), ".exe") {
			name = name[:len(name)-4]
		}
		if name != "" && entry.ProcessID != 0 {
			bytes, err := workingSet(entry.ProcessID)
			if err == nil {
				list = append(list, psProcess{Name: name, PID: int(entry.ProcessID), RAM: float64(int64(float64(bytes)/1048576.0 + 0.5))})
			}
		}
		entry.Size = uint32(unsafe.Sizeof(syscall.ProcessEntry32{}))
		if e = syscall.Process32Next(h, &entry); e != nil {
			break
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].RAM > list[j].RAM })
	if len(list) > 35 {
		list = list[:35]
	}
	return list, nil
}

type memoryStatus struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}
type filetime struct {
	Low  uint32
	High uint32
}

func ft(t filetime) uint64 { return uint64(t.High)<<32 | uint64(t.Low) }
func cpuTicks() (uint64, uint64, error) {
	var idle, kernel, user filetime
	r, _, e := getSystemTimes.Call(uintptr(unsafe.Pointer(&idle)), uintptr(unsafe.Pointer(&kernel)), uintptr(unsafe.Pointer(&user)))
	if r == 0 {
		return 0, 0, e
	}
	return ft(idle), ft(kernel) + ft(user), nil
}
func systemMetrics() (map[string]any, error) {
	m := memoryStatus{Length: uint32(unsafe.Sizeof(memoryStatus{}))}
	r, _, e := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&m)))
	if r == 0 {
		return nil, e
	}
	idle1, t1, _ := cpuTicks()
	time.Sleep(150 * time.Millisecond)
	idle2, t2, _ := cpuTicks()
	cpu := float64(0)
	if t2 > t1 {
		cpu = 100 * (1 - float64(idle2-idle1)/float64(t2-t1))
	}
	if cpu < 0 {
		cpu = 0
	}
	if cpu > 100 {
		cpu = 100
	}
	return map[string]any{"total_ram_bytes": m.TotalPhys, "available_ram_bytes": m.AvailPhys, "used_ram_bytes": m.TotalPhys - m.AvailPhys, "ram_percent": m.MemoryLoad, "cpu_percent": int(cpu + 0.5), "platform": "windows", "observed_at": time.Now().Format(time.RFC3339)}, nil
}
func runCmd(timeout time.Duration, executable string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c := exec.CommandContext(ctx, executable, args...)
	hideConsole(c)
	c.Env = append(os.Environ(), "GIT_PAGER=cat", "GIT_TERMINAL_PROMPT=0", "NO_COLOR=1")
	b, e := c.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", errors.New("comando excedeu tempo limite")
	}
	if len(b) > 20000 {
		b = b[:20000]
	}
	if e != nil {
		return "", fmt.Errorf("comando falhou: %v (%s)", e, clip(string(b), 500))
	}
	return string(b), nil
}

type psProcess struct {
	Name string  `json:"ProcessName"`
	PID  int     `json:"Id"`
	RAM  float64 `json:"RamMB"`
}

// A lista de processos e obtida pelas APIs do Windows, sem abrir PowerShell
// nem um console a cada atualizacao de 7 segundos.
func processList() ([]psProcess, error) {
	return runningProcesses()
}
func processName(pid int) (string, error) {
	if pid < 4 {
		return "", errors.New("PID invalido")
	}
	h, e := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if e != nil {
		return "", e
	}
	defer syscall.CloseHandle(h)
	entry := syscall.ProcessEntry32{Size: uint32(unsafe.Sizeof(syscall.ProcessEntry32{}))}
	if e = syscall.Process32First(h, &entry); e != nil {
		return "", e
	}
	for {
		if int(entry.ProcessID) == pid {
			name := syscall.UTF16ToString(entry.ExeFile[:])
			if strings.HasSuffix(strings.ToLower(name), ".exe") {
				name = name[:len(name)-4]
			}
			return name, nil
		}
		entry.Size = uint32(unsafe.Sizeof(syscall.ProcessEntry32{}))
		if e = syscall.Process32Next(h, &entry); e != nil {
			break
		}
	}
	return "", errors.New("processo nao encontrado")
}

var closable = map[string]bool{"chrome": true, "msedge": true, "firefox": true, "opera": true, "discord": true, "spotify": true, "teams": true, "slack": true, "telegram": true, "steam": true, "notion": true, "brave": true, "zoom": true}

func validateProcess(p ProcessTarget) error {
	if p.PID < 4 || p.PID > 2000000 {
		return errors.New("pid invalido")
	}
	n, e := processName(p.PID)
	if e != nil {
		return e
	}
	if !closable[strings.ToLower(n)] {
		return fmt.Errorf("aplicativo protegido ou nao autorizado: %s", n)
	}
	if !strings.EqualFold(n, p.Name) {
		return errors.New("nome de processo mudou; abortado")
	}
	return nil
}
func closeProcess(p ProcessTarget) error {
	if e := validateProcess(p); e != nil {
		return e
	}
	_, e := runCmd(7*time.Second, "taskkill.exe", "/PID", strconv.Itoa(p.PID))
	return e
}

var guidRE = regexp.MustCompile(`(?i)[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`)

func activePower() (string, error) {
	s, e := runCmd(8*time.Second, "powercfg.exe", "/getactivescheme")
	if e != nil {
		return "", e
	}
	return guidRE.FindString(s), nil
}
func highPower() (string, error) {
	s, e := runCmd(8*time.Second, "powercfg.exe", "/list")
	if e != nil {
		return "", e
	}
	const high = "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c"
	if strings.Contains(strings.ToLower(s), high) {
		return high, nil
	}
	return "", errors.New("plano Alto desempenho indisponivel neste computador; nao sera criado automaticamente")
}
func activatePower(guid string) error {
	if !guidRE.MatchString(guid) || len(guid) != 36 {
		return errors.New("GUID invalido")
	}
	_, e := runCmd(8*time.Second, "powercfg.exe", "/setactive", guid)
	return e
}
func boosterSession() map[string]any {
	p := filepath.Join(os.Getenv("LOCALAPPDATA"), "GameBoosterV2", "restauracao.json")
	b, e := os.ReadFile(p)
	if e != nil {
		return map[string]any{"external_booster_session": false}
	}
	var v map[string]any
	if json.Unmarshal(b, &v) != nil {
		return map[string]any{"external_booster_session": true, "message": "arquivo externo invalido"}
	}
	return map[string]any{"external_booster_session": true, "original_power": v["original_power"], "boosted_power": v["boosted_power"], "message": "Restaure a sessao anterior no Game Booster v2.1 antes de usar a energia pelo MCP"}
}
func restrictedCommand(action, dir string) (string, error) {
	switch action {
	case "windows_version":
		return runCmd(4*time.Second, "cmd.exe", "/d", "/c", "ver")
	case "git_status", "git_branch", "git_log":
		if dir == "" {
			return "", errors.New("forneca pasta de projeto")
		}
		var args []string
		switch action {
		case "git_status":
			args = []string{"--no-pager", "-C", dir, "status", "--short", "--untracked-files=no"}
		case "git_branch":
			args = []string{"--no-pager", "-C", dir, "branch", "--show-current"}
		case "git_log":
			args = []string{"--no-pager", "-C", dir, "log", "-5", "--oneline"}
		}
		return runCmd(7*time.Second, "git.exe", args...)
	default:
		return "", errors.New("comando nao autorizado")
	}
}
func openDashboard(url string) {
	exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url).Start()
}
