package gost_helper

import (
	"os"
	"slices"
	"strings"
	"testing"
)

func TestStrategies(t *testing.T) {
	for strategy, groups := range map[string]int{"off": 1, "aggressive": 6, "balanced": 6, "compat": 5, "disorder": 1, "fake": 1, "oob": 1, "tlsrec": 1} {
		t.Run(strategy, func(t *testing.T) {
			args, err := antiDPIArgs(AntiDPIConfig{Strategy: strategy, TTL: 17, Timeout: 9}, 12345, "/tmp/protect_path")
			if err != nil {
				t.Fatal(err)
			}
			count := 0
			for i, arg := range args {
				if arg == "--proto" {
					count++
					if args[i+1] != "http,tls" {
						t.Fatal("missing protocol scope")
					}
				}
				if arg == "--ttl" && args[i+1] != "17" {
					t.Fatal("TTL override lost")
				}
				if len(arg) >= 7 && arg[:7] == "--auto=" && args[i+1] != "--proto" {
					t.Fatal("fallback scope lost")
				}
			}
			if count != groups {
				t.Fatalf("got %d groups, want %d", count, groups)
			}
			if !slices.Contains(args, "/tmp/protect_path") {
				t.Fatal("VPN protection missing")
			}
		})
	}
}

func TestInvalidConfiguration(t *testing.T) {
	for _, config := range []AntiDPIConfig{{Strategy: "unknown"}, {Strategy: "fake", TTL: -1}, {Strategy: "fake", TTL: 256}, {Strategy: "fake", Timeout: -1}, {Strategy: "fake", Timeout: 301}} {
		if _, err := antiDPIArgs(config, 1081, ""); err == nil {
			t.Fatalf("accepted invalid config: %+v", config)
		}
	}
	args, err := antiDPIArgs(AntiDPIConfig{Strategy: "balanced"}, 1081, "")
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(args, "--protect-path") {
		t.Fatal("VPN protection enabled outside VPN mode")
	}
}

func TestRejectExternalYAML(t *testing.T) {
	previous := os.Args
	defer func() { os.Args = previous }()
	for _, arg := range []string{"-C", "--C", "-C=custom.yaml", "--C=custom.yaml"} {
		os.Args = []string{"gost", arg}
		err := startAntiDPI(AntiDPIConfig{Strategy: "balanced"}, "", false)
		if err == nil || !strings.Contains(err.Error(), "CLI -F") {
			t.Fatalf("YAML option %q was not rejected: %v", arg, err)
		}
	}
}
