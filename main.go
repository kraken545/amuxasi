// Amuxasi — Multi-Agent Coding Dashboard
//
// This is a convenience entry point. For the full CLI, use:
//   go install ./cmd/amuxasi/...
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	// Try to find the amuxasi binary (from cmd/amuxasi)
	// Look in GOPATH/bin, then PATH
	binary := "amuxasi"
	if runtime.GOOS == "windows" {
		binary = "amuxasi.exe"
	}

	// Try PATH first
	if path, err := exec.LookPath(binary); err == nil {
		runAmuxasi(path)
		return
	}

	// Try common Go bin locations
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}
	goBin := filepath.Join(gopath, "bin", binary)
	if _, err := os.Stat(goBin); err == nil {
		runAmuxasi(goBin)
		return
	}

	// Try building it
	fmt.Fprintf(os.Stderr, "amuxasi not found. Install it:\n")
	fmt.Fprintf(os.Stderr, "  go install ./cmd/amuxasi/...\n")
	os.Exit(1)
}

func runAmuxasi(path string) {
	cmd := exec.Command(path, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
