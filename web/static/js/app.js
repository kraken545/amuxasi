// Amuxasi Web UI — Application JavaScript
// Odysseus-inspired design, self-hosted, privacy-first

// ── Auth token handling ──────────────────────────────────
function getAuthToken() {
  return localStorage.getItem('amuxasi_token') || '';
}

function setAuthToken(token) {
  if (token) {
    localStorage.setItem('amuxasi_token', token);
  } else {
    localStorage.removeItem('amuxasi_token');
  }
}

function authHeaders() {
  const token = getAuthToken();
  return token ? { 'Authorization': 'Bearer ' + token } : {};
}

// ── API Client ───────────────────────────────────────────
const API = {
  async get(path) {
    const res = await fetch(`/api${path}`, {
      headers: { ...authHeaders() },
    });
    if (res.status === 401) {
      const errData = await res.json().catch(() => ({}));
      if (errData.error && errData.error.includes('authentication required')) {
        promptAuth();
        throw new Error('Authentication required');
      }
      throw new Error(errData.error || 'Unauthorized');
    }
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },
  async post(path, body) {
    const res = await fetch(`/api${path}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body: JSON.stringify(body),
    });
    if (res.status === 401) {
      const errData = await res.json().catch(() => ({}));
      if (errData.error && errData.error.includes('authentication required')) {
        promptAuth();
        throw new Error('Authentication required');
      }
      throw new Error(errData.error || 'Unauthorized');
    }
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },
};

function promptAuth() {
  const existing = getAuthToken();
  const token = prompt('🔐 Amuxasi requiere autenticación\nIngresa el token (AMUXASI_TOKEN):', existing);
  if (token !== null) {
    setAuthToken(token.trim());
    toast('Token guardado — recargando...');
    setTimeout(() => location.reload(), 500);
  }
}

// ---- Navigation ----
document.querySelectorAll('[data-section]').forEach(el => {
  el.addEventListener('click', e => {
    e.preventDefault();
    const section = el.dataset.section;
    showSection(section);
  });
});

function showSection(name) {
  document.querySelectorAll('.section').forEach(s => s.classList.remove('active'));
  document.querySelectorAll('[data-section]').forEach(s => s.classList.remove('active'));
  const sec = document.getElementById(`section-${name}`);
  if (sec) sec.classList.add('active');
  const nav = document.querySelector(`[data-section="${name}"]`);
  if (nav) nav.classList.add('active');
  window.location.hash = name;
}

// Hash-based routing
window.addEventListener('hashchange', () => {
  const hash = window.location.hash.slice(1) || 'dashboard';
  showSection(hash);
});

// Initial section from hash
const initHash = window.location.hash.slice(1) || 'dashboard';
showSection(initHash);

// ---- Toast ----
function toast(msg) {
  const t = document.getElementById('toast');
  t.textContent = msg;
  t.classList.add('show');
  setTimeout(() => t.classList.remove('show'), 3000);
}

// ---- Dashboard ----
async function loadDashboard() {
  try {
    const data = await API.get('/status');
    document.getElementById('stat-agents').textContent = data.agents?.length || 0;
    const running = data.agents?.filter(a => a.running).length || 0;
    document.getElementById('stat-running').textContent = running;
    document.getElementById('stat-detected').textContent = data.detected?.length || 0;
    document.getElementById('stat-tmux').textContent = data.hasTmux ? '✓' : '✗';
    document.getElementById('ws-name').textContent = data.workspace || 'workspace';
    document.getElementById('status-indicator').className = `dot ${running > 0 ? 'running' : 'stopped'}`;

    // Agent table
    const table = document.getElementById('agent-table');
    if (!data.agents || data.agents.length === 0) {
      table.innerHTML = '<div class="table-placeholder">No agents configured. Run <code>amuxasi init</code> or scan for detected agents.</div>';
      return;
    }
    table.innerHTML = data.agents.map(a => `
      <div class="agent-row">
        <div>
          <span class="agent-name">${a.name}</span>
          <span class="agent-status ${a.status}">${statusDot(a.status)} ${a.status}</span>
          <span class="ctx-bar"><span class="ctx-fill"><span class="ctx-fill-inner" style="width:${a.context || 50}%"></span></span><span class="ctx-pct">${a.context || 50}%</span></span>
        </div>
        <div class="agent-actions">
          ${a.running
            ? `<button class="btn danger" onclick="agentAction('${a.name}','stop')">Stop</button>
               <button class="btn" onclick="agentAction('${a.name}','attach')">Attach</button>`
            : `<button class="btn primary" onclick="agentAction('${a.name}','launch')">Launch</button>`
          }
        </div>
      </div>
    `).join('');
  } catch (err) {
    console.error('Dashboard load error:', err);
  }
}

function statusDot(status) {
  return `<span class="dot ${status}" style="display:inline-block;vertical-align:middle;margin-right:4px;"></span>`;
}

// Agent actions from dashboard
async function agentAction(name, action) {
  try {
    if (action === 'attach') {
      toast(`To attach: tmux attach-session -t amuxasi-${name}`);
      return;
    }
    const res = await API.post(`/agents/${name}/${action}`);
    toast(`${name}: ${res.status}`);
    loadDashboard();
    loadAgentsList();
  } catch (err) {
    toast(`Error: ${err.message}`);
  }
}

// ---- Chat ----
function sendChat() {
  const input = document.getElementById('chat-input');
  const msg = input.value.trim();
  if (!msg) return;

  const chat = document.getElementById('chat-messages');
  const placeholder = chat.querySelector('.chat-placeholder');
  if (placeholder) placeholder.remove();

  chat.innerHTML += `<div class="chat-msg"><div class="sender user">You</div><div class="text">${escapeHtml(msg)}</div></div>`;
  input.value = '';
  chat.scrollTop = chat.scrollHeight;

  // Send to API
  API.post('/debate/message', { message: msg }).then(res => {
    chat.innerHTML += `<div class="chat-msg"><div class="sender system">System</div><div class="text">${escapeHtml(res.message || 'Received')}</div></div>`;
    chat.scrollTop = chat.scrollHeight;
  }).catch(err => {
    chat.innerHTML += `<div class="chat-msg"><div class="sender system" style="color:var(--red)">Error</div><div class="text">${escapeHtml(err.message)}</div></div>`;
  });
}

function clearChat() {
  document.getElementById('chat-messages').innerHTML = '<div class="chat-placeholder">Chat cleared.</div>';
}

// Enter to send, Shift+Enter for newline
document.addEventListener('DOMContentLoaded', () => {
  const input = document.getElementById('chat-input');
  if (input) {
    input.addEventListener('keydown', e => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendChat();
      }
    });
  }
  const debateInput = document.getElementById('debate-chat-input');
  if (debateInput) {
    debateInput.addEventListener('keydown', e => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendDebateMessage();
      }
    });
  }
});

// ---- Debate ----
async function startDebate() {
  const topic = prompt('Tema del debate:') || 'General discussion';
  try {
    const res = await API.post('/debate', { action: 'start', topic });
    toast(`Debate started: ${topic}`);
    renderDebateState(res.state);
    loadDebate();
  } catch (err) {
    toast(`Error: ${err.message}`);
  }
}

async function stopDebate() {
  try {
    const res = await API.post('/debate', { action: 'stop' });
    toast('Debate stopped');
    renderDebateState(res.state);
    loadDebate();
  } catch (err) {
    toast(`Error: ${err.message}`);
  }
}

async function loadDebate() {
  try {
    const state = await API.get('/debate');
    renderDebateState(state);
  } catch (err) {
    console.error('Debate load error:', err);
  }
}

function renderDebateState(state) {
  if (!state) return;

  // Update debate info
  const info = document.getElementById('debate-info');
  const msgs = document.getElementById('debate-messages');
  const agents = document.getElementById('debate-agents');
  const consensus = document.getElementById('debate-consensus');

  if (info) {
    info.innerHTML = `
      <strong>Tema:</strong> ${state.topic || '(sin tema)'}<br>
      <strong>Estado:</strong> ${state.active ? '🟢 Activo' : '⏸️ Inactivo'}<br>
      <strong>Iniciado:</strong> ${state.started ? new Date(state.started).toLocaleString() : '-'}
    `;
  }

  // Messages
  if (msgs && state.messages) {
    if (state.messages.length === 0) {
      msgs.innerHTML = '<div class="chat-placeholder">No messages yet. Start a debate!</div>';
    } else {
      msgs.innerHTML = state.messages.slice(-50).map(m => {
        const isUser = m.sender === 'user';
        const isSystem = m.is_system;
        const roleColor = m.role ? getRoleColor(m.role) : 'var(--muted)';

        if (isSystem) {
          return `<div class="chat-msg system"><div class="text" style="color:var(--muted);font-style:italic;">${escapeHtml(m.text)}</div></div>`;
        }
        return `
          <div class="chat-msg ${isUser ? 'user' : 'agent'}">
            <div class="sender" style="color:${isUser ? 'var(--green)' : roleColor}">
              ${isUser ? 'You' : `${m.role_label || m.sender}: ${m.sender}`}
            </div>
            <div class="text">${escapeHtml(m.text)}</div>
          </div>`;
      }).join('');
      msgs.scrollTop = msgs.scrollHeight;
    }
  }

  // Agents
  if (agents && state.agents) {
    if (state.agents.length === 0) {
      agents.innerHTML = '<div class="table-placeholder">No agents in debate.</div>';
    } else {
      agents.innerHTML = state.agents.map(a => `
        <div class="agent-row">
          <div>
            <span class="agent-name">${a.agent_name}</span>
            <span style="color:${a.color};font-size:12px;">${a.role_label}</span>
          </div>
          <div>
            <span style="font-size:18px;">${a.vote_symbol}</span>
            <span class="ctx-bar" style="margin-left:8px;">
              <span class="ctx-fill"><span class="ctx-fill-inner" style="width:${a.context_pct}%"></span></span>
              <span class="ctx-pct">${a.context_pct}%</span>
            </span>
            <span style="color:var(--muted);font-size:11px;margin-left:8px;">${a.status_text || ''}</span>
          </div>
        </div>
      `).join('');
    }
  }

  // Consensus
  if (consensus && state.consensus) {
    const c = state.consensus;
    consensus.innerHTML = `
      <div style="display:flex;gap:16px;flex-wrap:wrap;">
        <div class="stat-card"><div class="stat-value">${c.consensus_pct || 0}%</div><div class="stat-label">Consenso</div></div>
        <div class="stat-card"><div class="stat-value" style="color:var(--green)">● ${c.agree_count || 0}</div><div class="stat-label">A favor</div></div>
        <div class="stat-card"><div class="stat-value" style="color:var(--red)">○ ${c.disagree_count || 0}</div><div class="stat-label">En contra</div></div>
        <div class="stat-card"><div class="stat-value" style="color:var(--yellow)">? ${c.confused_count || 0}</div><div class="stat-label">Confundidos</div></div>
        <div class="stat-card"><div class="stat-value">${c.avg_context_pct || 0}%</div><div class="stat-label">Contexto promedio</div></div>
      </div>
    `;
  }
}

function getRoleColor(role) {
  const colors = {
    'estratega': '#AA66FF',
    'critico': '#FF3333',
    'acelerador': '#00FF41',
    'disenador': '#00FFAA',
    'vigia': '#FF8800',
    'sintetizador': '#FFB000',
  };
  return colors[role] || '#CCCCCC';
}

// Debate actions from Dashboard
async function sendDebateMessage() {
  const input = document.getElementById('debate-chat-input');
  const msg = input.value.trim();
  if (!msg) return;

  input.value = '';
  try {
    const res = await API.post('/debate/message', { message: msg });
    renderDebateState(res.state);
    loadDebate();
  } catch (err) {
    toast(`Error: ${err.message}`);
  }
}

// ---- Agents ----
async function scanAgents() {
  try {
    const data = await API.get('/status');
    renderDetectedAgents(data.detected);
    toast('Scan complete');
  } catch (err) {
    toast(`Error: ${err.message}`);
  }
}

async function loadAgentsList() {
  try {
    const data = await API.get('/status');
    const list = document.getElementById('agent-list-detailed');
    if (!data.agents || data.agents.length === 0) {
      list.innerHTML = '<div class="table-placeholder">No agents configured.</div>';
    } else {
      list.innerHTML = data.agents.map(a => `
        <div class="agent-row">
          <div>
            <span class="agent-name">${a.name}</span>
            <span class="agent-source" style="color:var(--muted);font-size:12px;">(${a.source})</span>
            <span class="agent-status ${a.status}">${statusDot(a.status)} ${a.status}</span>
          </div>
          <div>
            <span style="color:var(--muted);font-size:12px;">${a.command}</span>
          </div>
          <div class="agent-actions">
            ${a.running
              ? `<button class="btn danger" onclick="agentAction('${a.name}','stop')">Stop</button>`
              : `<button class="btn primary" onclick="agentAction('${a.name}','launch')">Launch</button>`
            }
          </div>
        </div>
      `).join('');
    }

    renderDetectedAgents(data.detected);
  } catch (err) {
    console.error('Agents load error:', err);
  }
}

function renderDetectedAgents(detected) {
  const list = document.getElementById('detected-agents-list');
  if (!detected || detected.length === 0) {
    list.innerHTML = '<div class="table-placeholder">No agents detected on system.</div>';
    return;
  }
  list.innerHTML = `<div class="detected-grid">${detected.map(d => `
    <div class="detected-chip">
      <span class="check">✓</span>
      ${d.name}
      <span style="color:var(--muted);font-size:11px;">${d.path || ''}</span>
    </div>
  `).join('')}</div>`;
}

// ---- Settings ----
// Tabs
document.querySelectorAll('.setting-tab').forEach(tab => {
  tab.addEventListener('click', () => {
    document.querySelectorAll('.setting-tab').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('.setting-panel').forEach(p => p.classList.remove('active'));
    tab.classList.add('active');
    const panel = document.getElementById(`settings-${tab.dataset.tab}`);
    if (panel) panel.classList.add('active');
  });
});

// Theme
document.querySelectorAll('.theme-option').forEach(opt => {
  opt.addEventListener('click', () => {
    document.querySelectorAll('.theme-option').forEach(o => o.classList.remove('active'));
    opt.classList.add('active');
    document.documentElement.setAttribute('data-theme', opt.dataset.theme);
    localStorage.setItem('amuxasi-theme', opt.dataset.theme);
    toast(`Theme: ${opt.dataset.theme}`);
  });
});

// Load saved theme
const savedTheme = localStorage.getItem('amuxasi-theme');
if (savedTheme) {
  document.documentElement.setAttribute('data-theme', savedTheme);
  document.querySelectorAll('.theme-option').forEach(o => {
    o.classList.toggle('active', o.dataset.theme === savedTheme);
  });
}

// API Keys
async function loadKeys() {
  try {
    const keys = await API.get('/keys');
    const list = document.getElementById('keys-list');
    if (!keys || keys.length === 0) {
      list.innerHTML = '<div class="table-placeholder">No API keys configured.</div>';
      return;
    }
    list.innerHTML = keys.map(k => `
      <div class="key-row">
        <div>
          <span class="key-name">${k.name}</span>
          <span style="color:var(--muted);font-size:12px;margin-left:8px;">${k.set ? k.prefix : ''}</span>
        </div>
        <span class="key-status ${k.set ? 'set' : 'unset'}">${k.set ? '✓ Set' : '✗ Not set'}</span>
      </div>
    `).join('');
  } catch (err) {
    document.getElementById('keys-list').innerHTML = `<div class="table-placeholder">Error: ${err.message}</div>`;
  }
}

// Config
async function loadConfig() {
  try {
    const cfg = await API.get('/workspace');
    const editor = document.getElementById('config-editor');
    if (editor) {
      editor.value = JSON.stringify(cfg, null, 2);
    }
  } catch (err) {
    const editor = document.getElementById('config-editor');
    if (editor) editor.value = `Error: ${err.message}`;
  }
}

async function saveConfig() {
  // Placeholder - will implement config save with backend
  toast('Config save coming soon');
}

// ---- Helpers ----
function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

// ---- Health / Connection Monitor ----
let lastHealthOk = true;
let wasDown = false;

async function checkHealth() {
  try {
    const res = await fetch('/api/health');
    if (res.ok) {
      if (wasDown) {
        // El servidor se recuperó de una caída
        toast('🔄 Servidor reconectado');
        // Recargar datos completos
        loadDashboard();
        loadAgentsList();
        loadKeys();
        loadConfig();
        document.getElementById('connection-status').textContent = 'Connected';
        document.getElementById('connection-status').className = 'status-ok';
      }
      wasDown = false;
      lastHealthOk = true;
      document.getElementById('status-indicator').className = 'dot running';
    } else {
      throw new Error('Health check failed');
    }
  } catch (err) {
    if (lastHealthOk) {
      // Primera vez que falla
      wasDown = true;
      toast('⚠️ Conexión perdida — reintentando...');
      document.getElementById('connection-status').textContent = 'Disconnected';
      document.getElementById('connection-status').className = 'status-err';
    }
    lastHealthOk = false;
    document.getElementById('status-indicator').className = 'dot stopped';
  }
}

// ---- Connection Status in Nav ----
// Add connection status element after ws-name
document.addEventListener('DOMContentLoaded', () => {
  const statusEl = document.querySelector('.nav-status');
  if (statusEl && !document.getElementById('connection-status')) {
    const conn = document.createElement('span');
    conn.id = 'connection-status';
    conn.className = 'status-ok';
    conn.textContent = 'Connected';
    conn.style.cssText = 'font-size:11px;color:var(--muted);margin-left:8px;';
    statusEl.appendChild(conn);
  }
});

// ---- Polling ----
let pollInterval;

function startPolling() {
  loadDashboard();
  loadAgentsList();
  loadKeys();
  loadConfig();
  checkHealth();
  pollInterval = setInterval(() => {
    loadDashboard();
    loadAgentsList();
    loadDebate();
    checkHealth();
  }, 3000);
}

function stopPolling() {
  if (pollInterval) clearInterval(pollInterval);
}

// ---- Auth Check on Load ----
function checkAuthOnLoad() {
  const meta = document.querySelector('meta[name="amuxasi-auth"]');
  if (meta && meta.getAttribute('content') === 'true') {
    const token = getAuthToken();
    if (!token) {
      promptAuth();
    }
  }
}

// ---- Init ----
document.addEventListener('DOMContentLoaded', () => {
  checkAuthOnLoad();
  startPolling();
  // Cleanup on page unload
  window.addEventListener('beforeunload', stopPolling);
});
