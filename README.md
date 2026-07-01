# Amuxasi — Multi-Agent Coding Dashboard

> Dashboard retro terminal para ejecutar agentes de código lado a lado, con debate multi-agente integrado.
> **Self-hosted · Privacy-first · Sin telemetría · Trae tus propias API keys**

```
┌──────────────────────────────────────────────────────────┐
│  AMUXASI  ◆  mi-proyecto  [Agents]  Agents: 2/3  Ready  │
├────────────────────────────┬─────────────────────────────┤
│  ┌──────────────────────┐  │  Chat / Debate              │
│  │  Agentes             │  │                             │
│  │  ▸ ● [EST] claude 75%│  │  [EST] claude > Propongo    │
│  │    ○ [ACE] opencode  │  │  usar GraphQL porque...     │
│  │    ● [CRI] codex 50% │  │                             │
│  ├──────────────────────┤  │  [CRI] codex > ¿Has         │
│  │  Stats │ Topics │ ⚙  │  │  considerado REST?          │
│  └──────────────────────┘  │                             │
├────────────────────────────┴─────────────────────────────┤
│  ████████████████░░░░ 75%  ●2 ○1 ?0  🧠 68%  │  🍩 ●67% │
│  Tab:Next  i:Chat  l:Launch  s:Stop  b:Sidebar  q:Quit   │
└──────────────────────────────────────────────────────────┘
```

---

## 📦 Instalación

> **Filosofía:** Mínimos pasos para el usuario. Todo automatizado. Solo lo crucial es manual (API keys, configuración personal).

### 🐳 Opción 1: Docker (recomendado — 0 configuración)

**Un solo comando y ya está corriendo:**

```bash
git clone https://github.com/kraken545/amuxasi.git
cd amuxasi
docker compose up -d
# → http://localhost:7000
```

Para especificar API keys (opcional, solo si usas agentes cloud):

```bash
ANTHROPIC_API_KEY="sk-ant-..." docker compose up -d
# o edita docker-compose.yml y agrega tus keys
```

El contenedor incluye tmux y git, necesarios para los agentes.

### Construir la imagen manualmente

```bash
docker build -t amuxasi .
docker run -d --name amuxasi -p 7000:7000 \
  -v /ruta/de/tu/proyecto:/workspace \
  amuxasi
```

### 📦 Opción 2: Go install (CLI nativa)

Requiere Go ≥ 1.21 y tmux ≥ 3.3 instalados en tu sistema.

```bash
# CLI + Web
go install github.com/kraken545/amuxasi/cmd/amuxasi@latest

# TUI (requiere terminal interactiva)
go install github.com/kraken545/amuxasi/cmd/amuxasi-tui@latest

# Verificar
amuxasi version
# → amuxasi v0.2.0
```

### 🔧 Opción 3: Compilar desde fuente

```bash
git clone https://github.com/kraken545/amuxasi.git
cd amuxasi
go build -o amuxasi ./cmd/amuxasi/
go build -o amuxasi-tui ./cmd/amuxasi-tui/
sudo mv amuxasi amuxasi-tui /usr/local/bin/
```

---

## 🚀 Primeros pasos

### ⚡ La forma más rápida (recomendada)

```bash
git clone https://github.com/kraken545/amuxasi.git
cd amuxasi
docker compose up -d
# Abre http://localhost:7000 — 🎉 listo
```

Sin instalar nada más que Docker.

### 🖥️ Usar el dashboard TUI (terminal)

```bash
# 1. Inicializar en tu proyecto
cd /ruta/de/tu/proyecto
amuxasi init

# 2. Abrir el dashboard
amuxasi

# 3. Presiona 'l' para lanzar un agente
```

---

## 🎮 Dashboard completo

### Layout

```
┌────────────────────────────────────────────────────────────┐
│  Status Bar                                                │
│  [AMUXASI ◆ proyecto]  [Agents]  [Agents: 2/3]  [Ready]   │
├───────────────────────────┬────────────────────────────────┤
│  ┌─────────────────────┐  │  ┌──────────────────────────┐ │
│  │  Agent List         │  │  │  Chat / Debate           │ │
│  │  ▸ ● [EST] claude   │  │  │  Mensajes en vivo       │ │
│  │    ○ [ACE] opencode │  │  │  con colores por rol     │ │
│  │    ● [CRI] codex    │  │  │                          │ │
│  │  (contexto: 75%)    │  │  │  [input] > _             │ │
│  ├─────────────────────┤  │  └──────────────────────────┘ │
│  │  Sidebar (Stats/    │  │                               │
│  │   Topics/Config/    │  │                               │
│  │   Agents/Keys)      │  │                               │
│  └─────────────────────┘  └──────────────────────────────┘ │
├────────────────────────────────────────────────────────────┤
│  ████████████████░░░░ 75%  ●2 ○1 ?0  🧠 68%  │  🍩 ●67%  │
│  [Termómetro consenso]  [Votos]  [Contexto]  [Donut]      │
├────────────────────────────────────────────────────────────┤
│  Tab:Next  i:Chat  l:Launch  s:Stop  b:Sidebar  ?:Help    │
└────────────────────────────────────────────────────────────┘
```

### Atajos de teclado

#### Navegación

| Tecla | Acción | Contexto |
|---|---|---|
| `Tab` | Ciclar secciones | Agents → Chat → Sidebar |
| `Shift+Tab` | Ciclar reverso | Sidebar → Chat → Agents |
| `↑` / `k` | Arriba | Lista de agentes |
| `↓` / `j` | Abajo | Lista de agentes |
| `Enter` | Seleccionar / Enviar | General |

#### Control de agentes

| Tecla | Acción | Descripción |
|---|---|---|
| `l` | Lanzar | Inicia el agente en su propia sesión tmux |
| `s` | Detener | Mata el proceso del agente |
| `r` | Reiniciar | Detiene y vuelve a lanzar |
| `a` | Adjuntar | Te conecta a la sesión tmux del agente (Ctrl+B d para volver) |
| `?` | Diagnosticar | Abre las 5 preguntas si el agente está confundido (< 70%) |

#### Chat / Debate

| Tecla | Acción | Descripción |
|---|---|---|
| `i` | Modo input | Activa el campo de escritura de mensajes |
| `Enter` | Enviar | Envía el mensaje al chat |
| `D` | Iniciar debate | Activa el debate multi-agente sobre el tema actual |
| `X` | Detener debate | Finaliza el debate activo |

#### Sidebar

| Tecla | Acción | Descripción |
|---|---|---|
| `b` | Toggle | Muestra/oculta la barra lateral |
| `]` | Siguiente tab | Stats → Topics → Config → Agents → Keys |
| `[` | Anterior tab | Keys → Agents → Config → Topics → Stats |

#### Display

| Tecla | Acción |
|---|---|
| `Ctrl+L` | Toggle panel de logs |
| `F1` / `?` | Toggle ayuda completa |

#### Sesión

| Tecla | Acción |
|---|---|
| `d` | Desconectar — los agentes siguen corriendo en tmux, sales del dashboard |
| `q` / `Ctrl+C` | Salir del dashboard |

#### Scripts

| Tecla | Acción |
|---|---|
| `S` | Ejecutar script de setup (con confirmación de seguridad) |
| `A` | Ejecutar script de archive (con confirmación de seguridad) |

---

## 🤖 Agentes

### Agentes locales (auto-detectados)

Amuxasi detecta automáticamente estos agentes si están en tu `$PATH`:

| Agente | Comando | Rol por defecto |
|---|---|---|
| **Claude Code** | `claude` | 🧠 Estratega |
| **OpenCode** | `opencode` | ⚡ Acelerador |
| **Codex** | `codex` | 🔍 Crítico |
| **Gemini CLI** | `gemini` | 🧠 Estratega |
| **Amp** | `amp` | 🎨 Diseñador |
| **Droid** | `droid` | 🛡️ Vigía |
| **Aide** | `aide` | 🤖 Sintetizador |
| **Copilot** | `copilot` | ⚡ Acelerador |

### Agentes cloud (API)

Configura tus propias API keys como variables de entorno:

```bash
# En tu ~/.bashrc o ~/.zshrc:
export ANTHROPIC_API_KEY="sk-ant-..."
export OPENAI_API_KEY="sk-..."
export GEMINI_API_KEY="..."
export OPENROUTER_API_KEY="..."
export MISTRAL_API_KEY="..."
```

Y en tu `amuxasi.toml`:

```toml
[agents.claude-api]
provider = "anthropic"
api_key_env = "ANTHROPIC_API_KEY"
model = "claude-sonnet-4-20250514"

[agents.openrouter]
provider = "openrouter"
api_key_env = "OPENROUTER_API_KEY"
model = "mistralai/mixtral-8x7b"

[agents.gemini-api]
provider = "google"
api_key_env = "GEMINI_API_KEY"
model = "gemini-2.5-pro"
```

> **Filosofía:** Tú manages tus keys, no nosotros. Sin servidores intermedios, sin telemetría, sin recolectar datos. Cada agente cloud se conecta directamente a la API que le configures.

### Agentes personalizados

Puedes agregar cualquier comando como agente:

```toml
[agents.mi-script]
command = "python"
args = ["mi_agente.py", "--verbose"]
env = { PYTHONUNBUFFERED = "1" }
```

---

## 💬 Debate Multi-Agente

### ¿Qué es?

El debate multi-agente permite que varios agentes discutan un tema automáticamente, con roles especializados, votación silenciosa y un medidor de consenso en vivo.

### Roles

| Rol | Color | Personalidad | Enfoque |
|---|---|---|---|
| 🧠 **Estratega** | Púrpura | Visión global, piensa en grande | Arquitectura, roadmap, trade-offs |
| 🔍 **Crítico** | Rojo | Escéptico constructivo, busca fallos | Code review, edge cases |
| ⚡ **Acelerador** | Verde | Pragmático, acción rápida | MVP, soluciones rápidas |
| 🎨 **Diseñador** | Cian | Empático, pensamiento visual | UX, accesibilidad |
| 🛡️ **Vigía** | Naranja | Cauteloso, orientado a riesgos | Seguridad, estabilidad |
| 🤖 **Sintetizador** | Ámbar | Neutral, objetivo | Resume, concluye |

### Cómo usar

1. Escribe un tema en el chat (`i` para escribir, `Enter` para enviar)
2. Presiona `D` para iniciar el debate
3. Los agentes responden automáticamente con su perspectiva
4. El **termómetro de consenso** en la parte inferior muestra el progreso
5. El **donut chart** muestra el % de acuerdo/desacuerdo
6. Presiona `X` para detener el debate cuando quieras

### Votación silenciosa

Cada agente vota silenciosamente sin saber que lo ves:

| Símbolo | Significado |
|---|---|
| `●` | A favor (de acuerdo con la dirección) |
| `○` | En contra (cree que hay problema) |
| `?` | Confundido (no entiende el contexto) |
| `~` | Reformulando (reconsiderando su postura) |
| `·` | Esperando (no ha opinado aún) |

### Medidor de contexto

Cada agente tiene un medidor de contexto (0-100%) que indica qué tan informado está sobre el tema actual. Se actualiza automáticamente basado en:
- Participación en rondas de debate
- Preguntas de diagnóstico respondidas
- Ajuste manual del usuario

### Diagnóstico de 5 preguntas

Cuando un agente está por debajo del 70% de contexto, puedes presionar `?` para abrir un panel de diagnóstico. El agente te hará 5 preguntas de opción múltiple:

1. *"¿Qué archivo del proyecto necesito revisar?"*
2. *"¿Hay decisiones ya tomadas que deba considerar?"*
3. *"¿Cuál es el objetivo principal?"*
4. *"¿Hay restricciones técnicas?"*
5. *"¿Hay algo que ya funcione bien y no quieras cambiar?"*

Responde con los números `1-4` según la opción. Al completar las 5, el agente genera un **mini-informe** y retoma el debate con ~85% de contexto.

---

## 📊 Sidebar

Presiona `b` para abrir la barra lateral. Navega entre pestañas con `]` y `[`.

### Stats
Estadísticas en vivo del debate:
- Estado del debate (Activo/Inactivo)
- Tema actual
- Total de mensajes
- Número de agentes participantes
- Porcentaje de consenso
- Contexto promedio

### Topics
Temas activos del debate. Muestra el tema actual y permite cerrarlo.

### Config
Configuración actual del dashboard:
- Tema visual (Retro Terminal)
- Layout
- Estado de la sidebar

### Agents
Roles de cada agente en el debate activo con su nivel de contexto.

### Keys
Guía rápida para configurar API keys. Muestra las variables de entorno necesarias.

---

## 🔒 Trust System (Seguridad de Scripts)

Cuando ejecutas un script (`setup` o `archive`) por primera vez, Amuxasi muestra el contenido y pide aprobación:

```
┌────────────────────────────────────────────────────────────┐
│  Script: scripts/setup.sh                                  │
│                                                            │
│  #!/bin/sh                                                 │
│  npm install                                               │
│  ...                                                       │
│                                                            │
│  (y) Trust and run   (v) View full   (n/N) Reject         │
└────────────────────────────────────────────────────────────┘
```

- `y` → Aprueba y ejecuta
- `v` → Ver el script completo
- `n` / `N` / `esc` → Rechazar

**Cómo funciona:**
1. Se calcula el SHA256 del archivo de script
2. Se guarda el hash en `~/.config/amuxasi/trust.json`
3. Si el archivo cambia, se vuelve a pedir aprobación
4. Para revocar: borra el archivo `~/.config/amuxasi/trust.json` o la entrada correspondiente

---

## 📁 Configuración (`amuxasi.toml`)

### Archivo completo de ejemplo

```toml
# ============================================
# Amuxasi Configuration — amuxasi.toml
# ============================================

[workspace]
name = "mi-proyecto"
description = "API REST en Go con PostgreSQL"

# ============================================
# Agentes
# ============================================

# Agentes a lanzar por defecto al abrir el dashboard
launch = ["claude", "opencode"]

# Agente local: Claude Code
[agents.claude]
command = "claude"
args = ["--print", "--verbose"]
env = { ANTHROPIC_API_KEY = "${ANTHROPIC_API_KEY}" }

# Agente local: OpenCode
[agents.opencode]
command = "opencode"
args = []

# Agente cloud: Claude API
[agents.claude-api]
provider = "anthropic"
api_key_env = "ANTHROPIC_API_KEY"
model = "claude-sonnet-4-20250514"

# Agente cloud: OpenRouter (acceso a modelos open-source)
[agents.openrouter]
provider = "openrouter"
api_key_env = "OPENROUTER_API_KEY"
model = "mistralai/mixtral-8x7b"

# ============================================
# Roles del debate
# ============================================

[[debate.agents]]
name = "claude"
rol = "estratega"
personalidad = "Piensa en grande, sugiere arquitecturas robustas"

[[debate.agents]]
name = "opencode"
rol = "critico"
personalidad = "Encuentra fallos, cuestiona supuestos"

# Roles disponibles:
#   estratega    — Visión global, arquitectura
#   critico      — Code review, edge cases
#   acelerador   — Pragmático, acción rápida
#   disenador    — UX, accesibilidad
#   vigia        — Seguridad, monitoreo
#   sintetizador — Conclusión, resumen

# ============================================
# Scripts
# ============================================

[scripts]
setup = "scripts/setup.sh"
run = "scripts/run.sh"
archive = "scripts/archive.sh"
```

---

## 📂 Git Worktrees

Los worktrees permiten tener múltiples branches del mismo repo abiertas simultáneamente, cada una con sus propios agentes.

```bash
# Crear un worktree con su propio amuxasi.toml
amuxasi add-worktree ../feature-xyz feature-branch

# Navegar al worktree y abrir el dashboard
cd ../feature-xyz
amuxasi
```

Esto crea:
- El git worktree en `../feature-xyz`
- Un `amuxasi.toml` dentro del worktree
- El workspace se nombra automáticamente como la branch

---

## 📝 Comandos CLI

| Comando | Descripción |
|---|---|
| `amuxasi` | Abre el dashboard TUI (requiere `amuxasi-tui`) |
| `amuxasi web` | Inicia la Web UI en http://localhost:7000 |
| `amuxasi init` | Crea `amuxasi.toml` en el directorio actual |
| `amuxasi add-worktree <path> [branch]` | Crea git worktree + configuración |
| `amuxasi archive` | Archiva el workspace actual |
| `amuxasi help` | Muestra la ayuda completa |
| `amuxasi version` | Muestra la versión |

---

## 🪵 Logs

Los logs se guardan en `~/.cache/amuxasi/logs/`:

```
~/.cache/amuxasi/logs/
├── amuxasi.log              # Eventos generales de la app
└── mi-proyecto/
    ├── claude.log            # Salida del agente Claude
    ├── opencode.log          # Salida del agente OpenCode
    └── codex.log             # Salida del agente Codex
```

Dentro del dashboard, presiona `Ctrl+L` para ver los logs en vivo.

---

## 🔐 Seguridad

### API Keys
- **Nunca** se hardcodean en el código
- **Nunca** se envían a servidores externos
- Se leen de variables de entorno (`$ANTHROPIC_API_KEY`, etc.)
- Cada agente cloud se conecta **directamente** a la API que configures
- Sin telemetría, sin recolectar datos, sin servidores intermedios

### Scripts
- Sistema de trust con SHA256
- El usuario debe aprobar explícitamente cada script
- Si el script cambia, se vuelve a pedir aprobación
- Los approvals se guardan localmente en `~/.config/amuxasi/trust.json`

### Aislamiento
- Cada agente corre en su propia sesión **tmux**
- Si cierras el dashboard, los agentes siguen corriendo
- Puedes reconectar con `amuxasi` y ver el estado

---

## 🌐 Web UI

Amuxasi incluye una Web UI estilo Odysseus (inspirada en PewDiePie) con panel de control, chat multi-agente, debate, y configuración. Todo en una SPA vanilla (sin React/Vue) incrustada en el binario Go via `embed.FS`.

### 🐳 Usar con Docker (automático)

```bash
# Un comando y ya:
docker compose up -d
# → http://localhost:7000

# Con API keys (opcional):
ANTHROPIC_API_KEY="sk-ant-..." docker compose up -d

# Ver logs:
docker compose logs -f

# Detener:
docker compose down
```

### Construir la imagen manualmente

```bash
make docker-build
# o directamente:
docker build -t amuxasi .
```

### 🖥️ Usar desde tu máquina (sin Docker)

```bash
amuxasi web
# → http://localhost:7000

# Puerto personalizado:
amuxasi web --port 8080
```

### Estructura del frontend

```
web/static/
├── index.html    # Punto de entrada SPA
├── style.css     # Tema oscuro Odysseus (CSS variables, Fira Code)
└── js/app.js     # Lógica: routing, API calls, polling, themes
```

### API REST

| Endpoint | Método | Descripción |
|---|---|---|
| `/api/health` | GET | Health check |
| `/api/status` | GET | Estado completo del sistema |
| `/api/agents` | GET | Lista de agentes |
| `/api/agents/launch` | POST | Lanzar agente |
| `/api/agents/stop` | POST | Detener agente |
| `/api/workspace` | GET | Info del workspace |
| `/api/debate` | GET/POST | Debate multi-agente |
| `/api/debate/message` | POST | Enviar mensaje al debate |
| `/api/keys` | GET | Muestra las variables de entorno disponibles |
| `/api/config` | GET | Configuración actual |
| `/api/logs` | GET | Logs en vivo |

---

## ⚙️ Solución de problemas

| Problema | Causa | Solución |
|---|---|---|
| `tmux not found` | tmux no instalado | `brew install tmux` / `sudo apt install tmux` |
| Agent not detected | No está en `$PATH` | Instala el agente o usa `command` con ruta completa |
| Config not loaded | No hay `amuxasi.toml` | `amuxasi init` |
| Not in a git repo | Solo necesario para worktrees | No es necesario para el dashboard básico |
| API key not working | Variable de entorno no definida | `export ANTHROPIC_API_KEY="sk-..."` |
| TUI no se abre | No hay TTY disponible | Ejecuta en una terminal real, no en CI |

---

## 🧪 Desarrollo

```bash
# Clonar
git clone https://github.com/kraken545/amuxasi.git
cd amuxasi

# Compilar (todos los binarios)
go build ./cmd/amuxasi/
go build ./cmd/amuxasi-tui/
go build ./cmd/web/

# Tests
go test ./...

# Vet
go vet ./...

# Instalar localmente
go install ./cmd/amuxasi/...
go install ./cmd/amuxasi-tui/...
```

### Estructura del proyecto

```
amuxasi/
├── main.go              # Punto de entrada (thin wrapper)
├── cmd/
│   ├── amuxasi/         # CLI principal (init, web, add-worktree, etc.)
│   │   └── main.go
│   ├── amuxasi-tui/     # Dashboard TUI (Bubble Tea)
│   │   └── main.go
│   └── web/             # Web server (Docker)
│       └── main.go
├── config/config.go     # Parseo de amuxasi.toml
├── agent/
│   ├── agent.go         # Ciclo de vida de agentes
│   └── tmux.go          # Wrapper de tmux
├── trust/trust.go       # Sistema de aprobación SHA256
├── workspace/           # Gestión de workspaces y worktrees
├── log/log.go           # Logging estructurado
├── web/                 # Servidor HTTP + API REST
│   ├── server.go        # Rutas, CORS, handler estático
│   ├── handlers.go      # Handlers de API
│   └── static/          # Frontend SPA
│       ├── index.html
│       ├── style.css
│       └── js/app.js
├── tui/
│   ├── tui.go           # Modelo principal Bubble Tea
│   ├── chat.go          # Debate multi-agente
│   ├── sidebar.go       # Barra lateral con pestañas
│   ├── styles.go        # Tema retro terminal
│   └── keys.go          # Definición de atajos
├── Dockerfile           # Multi-stage build (web server)
├── docker-compose.yml   # Docker Compose (web UI)
└── Makefile             # Automatización: make docker-up, make build, etc.
```

---

## 🗺️ Roadmap

| Feature | Estado |
|---|---|
| Dashboard TUI con split panes | ✅ |
| Agentes locales en tmux | ✅ |
| Auto-detección de agentes | ✅ |
| Trust system (SHA256) | ✅ |
| Git worktrees | ✅ |
| Debate multi-agente | ✅ |
| 5 preguntas de diagnóstico | ✅ |
| Medidor de consenso + donut | ✅ |
| Sidebar con pestañas | ✅ |
| API Keys (variables de entorno) | ✅ |
| **Web UI** | ✅ Completado |
| **Temas visuales intercambiables** | 🟡 Planeado |
| **Editor visual de configuración** | 🟡 Planeado |
| **Historial de sesiones** | 🔴 Futuro |
| **Agentes cloud (OpenAI, Anthropic, OpenRouter)** | 🔴 Futuro |
| **Instaladores (Homebrew, APT)** | 🔴 Futuro |

---

## 📜 Licencia

AGPL-3.0 — Ver archivo [LICENSE](LICENSE) para más detalles.

---

## 🙏 Inspiración

- **Odysseus** de PewDiePie — diseño self-hosted, privacy-first, filosofía "trae tus propias keys"
- **Bubble Tea** — framework TUI en Go
- **tmux** — multiplexor de terminal
