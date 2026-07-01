# Amuxasi — Makefile
# Automatiza todo: build, run, docker, clean
# Filosofía: menos pasos para el usuario

.PHONY: all build build-tui build-web docker docker-build docker-up docker-down run web clean help

all: help

# ─── Binarios ───────────────────────────────────────────────

build: ## Compilar todos los binarios localmente
	go build -o amuxasi ./cmd/amuxasi/
	go build -o amuxasi-tui ./cmd/amuxasi-tui/
	go build -o amuxasi-web ./cmd/web/

build-cli: ## Compilar solo el CLI
	go build -o amuxasi ./cmd/amuxasi/

build-tui: ## Compilar solo la TUI
	go build -o amuxasi-tui ./cmd/amuxasi-tui/

build-web: ## Compilar solo el web server
	go build -o amuxasi-web ./cmd/web/

# ─── Docker ─────────────────────────────────────────────────

docker: docker-build ## Build y arrancar en un solo paso (alias)
docker-build: ## Construir la imagen Docker
	docker build -t amuxasi .

docker-up: ## Arrancar con docker compose
	docker compose up -d

docker-down: ## Detener docker compose
	docker compose down

docker-logs: ## Ver logs del contenedor
	docker compose logs -f

docker-restart: docker-down docker-up ## Reiniciar contenedor

# ─── Ejecución ──────────────────────────────────────────────

run: ## Arrancar Web UI localmente (puerto 7000)
	go run ./cmd/web/ --port 7000 --workspace .

run-web: run ## Alias para arrancar Web UI

run-cli: ## Arrancar CLI (sin TUI, solo comandos)
	go run ./cmd/amuxasi/ help

# ─── Utilidades ─────────────────────────────────────────────

clean: ## Limpiar binarios compilados
	rm -f amuxasi amuxasi-tui amuxasi-web

test: ## Ejecutar tests
	go test ./...

vet: ## Verificar código
	go vet ./...

help: ## Mostrar esta ayuda
	@echo "Amuxasi — Comandos disponibles"
	@echo "────────────────────────────────"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Ejemplos rápidos:"
	@echo "  make docker-up      → http://localhost:7000"
	@echo "  make run            → http://localhost:7000 (local)"
	@echo "  make build          → compilar todo"
