package gost_helper

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSupervisorHelper(t *testing.T) {
	if os.Getenv("ANTIDPI_SUPERVISOR_HELPER") != "1" {
		return
	}
	os.Args = []string{os.Args[0], "-L", "ss://none@127.0.0.1:9999"}
	if err := startAntiDPI(AntiDPIConfig{Strategy: "balanced"}, "", false); err != nil {
		t.Fatal(err)
	}
	fmt.Println("SUPERVISOR_READY " + os.Args[2])
	select {}
}

func TestSupervisorLifecycle(t *testing.T) {
	engine := os.Getenv("BYEDPI_TEST_ENGINE")
	if engine == "" {
		t.Skip("set BYEDPI_TEST_ENGINE to a host ByeDPI executable")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(engine)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(filepath.Dir(executable), "libbyedpi.so"), data, 0700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(executable, "-test.run=^TestSupervisorHelper$")
	cmd.Env = append(os.Environ(), "ANTIDPI_SUPERVISOR_HELPER=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	ready := make(chan string, 1)
	go func() {
		scan := bufio.NewScanner(stdout)
		for scan.Scan() {
			if line := scan.Text(); strings.HasPrefix(line, "SUPERVISOR_READY ") {
				ready <- strings.TrimPrefix(line, "SUPERVISOR_READY ")
				return
			}
		}
		ready <- ""
	}()
	var endpoint string
	select {
	case endpoint = <-ready:
	case <-time.After(10 * time.Second):
		t.Fatal("supervisor startup timed out")
	}
	if endpoint == "" {
		t.Fatal("supervisor failed before ready")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.DialTimeout("tcp", parsed.Host, time.Second)
	if err != nil {
		t.Fatal("engine not listening", err)
	}
	conn.Close()
	cmd.Process.Kill()
	cmd.Wait()
	for i := 0; i < 100; i++ {
		conn, err = net.DialTimeout("tcp", parsed.Host, 50*time.Millisecond)
		if err != nil {
			return
		}
		conn.Close()
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("ByeDPI survived the supervisor")
}
