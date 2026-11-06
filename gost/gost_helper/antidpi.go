package gost_helper

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

//go:embed strategies.json
var strategyJSON []byte

type AntiDPIConfig struct {
	Strategy string
	TTL      int
	Timeout  int
}

func antiDPIArgs(config AntiDPIConfig, port int, protectPath string) ([]string, error) {
	var presets map[string][][]string
	if err := json.Unmarshal(strategyJSON, &presets); err != nil {
		return nil, err
	}
	groups, ok := presets[config.Strategy]
	if !ok {
		return nil, fmt.Errorf("unknown strategy %q", config.Strategy)
	}
	if config.TTL == 0 {
		config.TTL = 8
	}
	if config.Timeout == 0 {
		config.Timeout = 3
	}
	if config.TTL < 1 || config.TTL > 255 || config.Timeout < 1 || config.Timeout > 300 {
		return nil, fmt.Errorf("TTL must be 1..255; timeout must be 1..300")
	}
	args := []string{"--ip", "127.0.0.1", "--port", strconv.Itoa(port), "--max-conn", "1024",
		"--timeout", strconv.Itoa(config.Timeout), "--cache-ttl", "86400", "--auto-mode", "3", "--no-udp", "--debug", "1"}
	if protectPath != "" {
		args = append(args, "--protect-path", protectPath)
	}
	for _, group := range groups {
		if len(group) > 0 && len(group[0]) >= 7 && group[0][:7] == "--auto=" {
			args = append(args, group[0])
			group = group[1:]
		}
		args = append(args, "--proto", "http,tls")
		for i, arg := range group {
			if i > 0 && group[i-1] == "--ttl" {
				arg = strconv.Itoa(config.TTL)
			}
			args = append(args, arg)
		}
	}
	return args, nil
}

// The engine is packaged beside GOST in nativeLibraryDir and runs under the
// Shadowsocks UID. Its upstream protect-path option uses the host VPN service.
func startAntiDPI(config AntiDPIConfig, protectPath string, vpn bool) error {
	if config.Strategy == "" || config.Strategy == "off" {
		return nil
	}
	// Only CLI chains can be safely amended. Do not silently bypass YAML chains.
	for _, arg := range os.Args[1:] {
		if arg == "-C" || arg == "--C" || strings.HasPrefix(arg, "-C=") || strings.HasPrefix(arg, "--C=") {
			return fmt.Errorf("AntiDPI requires a CLI -F chain, not -C")
		}
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	if !vpn {
		protectPath = ""
	}
	args, err := antiDPIArgs(config, port, protectPath)
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(filepath.Join(filepath.Dir(executable), "libbyedpi.so"), args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	// Called during Go init on the main OS thread. Keep that thread alive:
	// Linux ties Pdeathsig to the thread which starts the child.
	runtime.LockOSThread()
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
	if err = cmd.Start(); err != nil {
		return err
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	ready := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-exited:
			return fmt.Errorf("ByeDPI stopped during startup: %v", err)
		default:
		}
		conn, err := net.DialTimeout("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 100*time.Millisecond)
		if err == nil {
			conn.SetDeadline(time.Now().Add(200 * time.Millisecond))
			_, err = conn.Write([]byte{5, 1, 0})
			response := make([]byte, 2)
			if err == nil {
				_, err = io.ReadFull(conn, response)
			}
			conn.Close()
			if err == nil && response[0] == 5 && response[1] == 0 {
				ready = true
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !ready {
		cmd.Process.Kill()
		<-exited
		return fmt.Errorf("ByeDPI SOCKS5 startup timed out")
	}
	// GOST dials its first remote hop through the local AntiDPI SOCKS server.
	os.Args = append([]string{os.Args[0], "-F", fmt.Sprintf("socks5://127.0.0.1:%d", port)}, os.Args[1:]...)
	go func() {
		err := <-exited
		fmt.Fprintf(os.Stderr, "ByeDPI exited (%v); stopping GOST\n", err)
		os.Exit(1)
	}()
	return nil
}
