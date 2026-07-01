# Amuxasi — Makefile
# Automatiza todo: build, run, docker, dev, backup
# Filosofía: menos pasos para el usuario, siempre preguntando antes de acciones destructivas

.PHONY: all build build-tui build-web docker docker-build docker-up docker-down \
        run web dev setup update backup restore install clean test vet help

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

install: build ## Compilar e instalar en GOPATH/bin
	go install ./cmd/amuxasi/...
	go install ./cmd/amuxasi-tui/...
	@echo "✅ Instalado. Usa: amuxasi  o  amuxasi web"

# ─── Docker ─────────────────────────────────────────────────

docker: docker-build docker-up ## Build y arrancar en un solo paso (alias)

docker-build: ## Construir la imagen Docker
	docker build -t amuxasi .

docker-up: ## Arrancar con docker compose
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "📄 Creado .env desde .env.example (valores vacíos)"; \
		echo "   Para API keys: edita .env o usa: export KEY=valor docker compose up"; \
	fi
	docker compose up -d
	@echo "🌐 Amuxasi → http://localhost:7000"

docker-down: ## Detener docker compose
	docker compose down

docker-logs: ## Ver logs del contenedor
	docker compose logs -f

docker-restart: docker-down docker-up ## Reiniciar contenedor

docker-status: ## Ver estado del contenedor
	docker compose ps
	@echo "---"
	-docker inspect --format='{{.State.Health.Status}}' amuxasi 2>/dev/null || echo "Healthcheck: no disponible"

# ─── Ejecución ──────────────────────────────────────────────

run: ## Arrancar Web UI localmente (puerto 7000)
	go run ./cmd/web/ --port 7000 --workspace .

run-web: run ## Alias para arrancar Web UI

run-cli: ## Arrancar CLI (sin TUI, solo comandos)
	go run ./cmd/amuxasi/ help

dev: ## Arrancar con hot-reload (requiere air: go install github.com/air-verse/air@latest)
	@if ! command -v air >/dev/null 2>&1; then \
		echo "⚡ Hot-reload requiere 'air'. Instálalo con:"; \
		echo "   go install github.com/air-verse/air@latest"; \
		echo "   make dev  # de nuevo"; \
		exit 1; \
	fi
	air

# ─── Actualización ──────────────────────────────────────────

update: ## Actualizar Amuxasi (git pull + preguntar rebuild)
	@echo "🔄 Actualizando Amuxasi..."
	git pull
	@echo ""
	@printf "¿Recompilar binarios? (y/n): "; read ans; \
	if [ "$$ans" = "y" ] || [ "$$ans" = "Y" ]; then \
		$(MAKE) build; \
	fi
	@echo ""
	@printf "¿Reiniciar contenedor Docker? (y/n): "; read ans; \
	if [ "$$ans" = "y" ] || [ "$$ans" = "Y" ]; then \
		$(MAKE) docker-restart; \
	fi
	@echo "✅ Actualización completada"

# ─── Backup / Restore ───────────────────────────────────────

BACKUP_DIR = ./backups
BACKUP_FILE = $(BACKUP_DIR)/amuxasi-backup-$(shell date +%Y%m%d-%H%M%S).tar.gz

backup: ## Crear backup de configuración (amuxasi.toml + trust.json)
	@mkdir -p $(BACKUP_DIR)
	@echo "📦 Creando backup..."
	@config_dir=$${XDG_CONFIG_HOME:-$$HOME/.config}/amuxasi; \
	files=""; \
	if [ -f amuxasi.toml ]; then files="$$files amuxasi.toml"; fi; \
	if [ -f "$$config_dir/trust.json" ]; then files="$$files $$config_dir/trust.json"; fi; \
	if [ -z "$$files" ]; then \
		echo "⚠️  No hay archivos de configuración para respaldar."; \
		exit 0; \
	fi; \
	tar -czf $(BACKUP_FILE) $$files 2>/dev/null; \
	echo "✅ Backup creado: $(BACKUP_FILE)"

restore: ## Restaurar configuración desde un backup
	@latest=$$(ls -t $(BACKUP_DIR)/amuxasi-backup-*.tar.gz 2>/dev/null | head -1); \
	if [ -z "$$latest" ]; then \
		echo "⚠️  No hay backups en $(BACKUP_DIR)/"; \
		exit 0; \
	fi; \
	echo "📂 Backup más reciente: $$latest"; \
	printf "¿Restaurar? (y/n): "; read ans; \
	if [ "$$ans" = "y" ] || [ "$$ans" = "Y" ]; then \
		tar -xzf "$$latest" -C /; \
		echo "✅ Restaurado desde: $$latest"; \
	else \
		echo "Cancelado."; \
	fi

backup-list: ## Listar backups disponibles
	@echo "📂 Backups disponibles:"
	@ls -lh $(BACKUP_DIR)/amuxasi-backup-*.tar.gz 2>/dev/null || echo "   (ninguno)"

# ─── Utilidades ─────────────────────────────────────────────

setup: ## Mostrar dependencias necesarias y cómo instalarlas
	@echo "📋 Amuxasi — Dependencias"
	@echo ""
	@if command -v tmux >/dev/null 2>&1; then \
		echo "  ✅ tmux:    $$(tmux -V)"; \
	else \
		echo "  ❌ tmux:    No instalado"; \
		echo "     macOS: brew install tmux"; \
		echo "     Linux: sudo apt install tmux  o  sudo pacman -S tmux"; \
	fi
	@if command -v git >/dev/null 2>&1; then \
		echo "  ✅ git:     $$(git --version)"; \
	else \
		echo "  ❌ git:     No instalado"; \
		echo "     macOS: brew install git"; \
		echo "     Linux: sudo apt install git  o  sudo pacman -S git"; \
	fi
	@if command -v go >/dev/null 2>&1; then \
		echo "  ✅ Go:      $$(go version)"; \
	else \
		echo "  ❌ Go:      No instalado (solo necesario para compilar desde fuente)"; \
		echo "     https://go.dev/dl/"; \
	fi
	@if command -v air >/dev/null 2>&1; then \
		echo "  ✅ air:     $$(air -v 2>&1 | head -1)"; \
	else \
		echo "  ⚡ air:     No instalado (opcional, para hot-reload)"; \
		echo "     go install github.com/air-verse/air@latest"; \
	fi
	@if command -v docker >/dev/null 2>&1; then \
		echo "  ✅ Docker:  $$(docker --version)"; \
	else \
		echo "  ❌ Docker:  No instalado (opcional, para despliegue web)"; \
		echo "     https://docs.docker.com/get-docker/"; \
	fi

clean: ## Limpiar binarios compilados
	rm -f amuxasi amuxasi-tui amuxasi-web
	rm -rf /tmp/amuxasi-air
	@echo "🧹 Limpiado"

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
	@echo "  make dev            → hot-reload (requiere air)"
	@echo "  make setup          → verificar dependencias"
	@echo "  make backup         → respaldar configuración"
	@echo "  make update         → actualizar Amuxasi"
