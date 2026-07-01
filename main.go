// Amuxasi — Multi-Agent Coding Dashboard
//
// Punto de entrada principal. Busca el binario amuxasi en:
//   1. PATH
//   2. GOPATH/bin
//   3. ./cmd/amuxasi/ (intenta compilar si no existe)
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	binary := "amuxasi"
	if runtime.GOOS == "windows" {
		binary = "amuxasi.exe"
	}

	// 1. Buscar en PATH
	if path, err := exec.LookPath(binary); err == nil {
		runAmuxasi(path)
		return
	}

	// 2. Buscar en GOPATH/bin
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

	// 3. Buscar en el directorio actual
	localBin := filepath.Join(".", binary)
	if _, err := os.Stat(localBin); err == nil {
		runAmuxasi(localBin)
		return
	}

	// 4. No encontrado: preguntar si compilar
	fmt.Println("📦 Amuxasi no está instalado.")
	if askYesNo("¿Compilar amuxasi ahora (requiere Go)?") {
		if compileAmuxasi() {
			// Reintentar ejecutar
			if path, err := exec.LookPath(binary); err == nil {
				runAmuxasi(path)
				return
			}
			if _, err := os.Stat(localBin); err == nil {
				runAmuxasi(localBin)
				return
			}
		}
	}

	fmt.Fprintf(os.Stderr, "\nPara instalar manualmente:\n")
	fmt.Fprintf(os.Stderr, "  go install ./cmd/amuxasi/...\n")
	os.Exit(1)
}

func askYesNo(question string) bool {
	fmt.Printf("%s (y/n): ", question)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes" || answer == "s" || answer == "si"
}

func compileAmuxasi() bool {
	fmt.Println("⚙️  Compilando amuxasi...")

	cmd := exec.Command("go", "build", "-o", "amuxasi", "./cmd/amuxasi/")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error compilando: %v\n", err)
		fmt.Println("Asegúrate de tener Go instalado: https://go.dev/dl/")
		return false
	}

	// Mover a GOPATH/bin
	dest := "amuxasi"
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		dest = filepath.Join(gopath, "bin", "amuxasi")
		os.MkdirAll(filepath.Dir(dest), 0755)
		os.Rename("amuxasi", dest)
	} else if home, _ := os.UserHomeDir(); home != "" {
		dest = filepath.Join(home, "go", "bin", "amuxasi")
		os.MkdirAll(filepath.Dir(dest), 0755)
		os.Rename("amuxasi", dest)
	}

	fmt.Printf("✅ Compilado: %s\n", dest)
	return true
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
