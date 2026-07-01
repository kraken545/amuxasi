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

// ---- Compare ----
async function startCompare() {
  const label = document.getElementById('compare-label').value.trim();
  const prompt = document.getElementById('compare-prompt').value.trim();
  const timeout = parseInt(document.getElementById('compare-timeout').value) || 30;

  if (!prompt) {
    toast('Please enter a prompt');
    return;
  }

  // Get selected agents
  const checkboxes = document.querySelectorAll('#compare-agent-select input:checked');
  const agentNames = Array.from(checkboxes).map(cb => cb.value);

  if (agentNames.length === 0) {
    toast('Please select at least one agent');
    return;
  }

  try {
    const res = await API.post('/compare', {
      label,
      prompt,
      agent_names: agentNames,
      timeout,
    });
    toast(`Compare started: ${res.id}`);
    loadCompareSessions();
  } catch (err) {
    toast(`Error: ${err.message}`);
  }
}

async function loadCompareSessions() {
  try {
    const sessions = await API.get('/compare');
    const div = document.getElementById('compare-sessions');

    if (!sessions || sessions.length === 0) {
      div.innerHTML = '<div class="table-placeholder">No comparisons yet.</div>';
      return;
    }

    div.innerHTML = sessions.map(s => {
      const resultsHtml = (s.results || []).map(r => `
        <div class="compare-result" style="flex:1;min-width:250px;border:1px solid var(--border);border-radius:6px;padding:12px;background:var(--bg-secondary);">
          <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:8px;">
            <strong style="color:${r.status === 'complete' ? 'var(--green)' : r.status === 'error' ? 'var(--red)' : 'var(--yellow)'}">${escapeHtml(r.agent_name)}</strong>
            <span style="font-size:11px;color:var(--muted);">${r.duration || '...'}</span>
          </div>
          <div style="font-size:11px;color:var(--muted);margin-bottom:6px;">${r.status}</div>
          ${r.output ? `<pre style="font-size:12px;white-space:pre-wrap;word-break:break-word;max-height:300px;overflow-y:auto;margin:0;">${escapeHtml(truncStr(r.output, 1000))}</pre>` : ''}
          ${r.error ? `<div style="color:var(--red);font-size:12px;margin-top:4px;">⚠️ ${escapeHtml(r.error)}</div>` : ''}
          ${r.status === 'running' || r.status === 'pending' ? '<div style="color:var(--yellow);font-size:12px;">⏳ Running...</div>' : ''}
        </div>
      `).join('');

      return `
        <div class="card" style="margin-bottom:12px;">
          <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:8px;">
            <div>
              <strong>${escapeHtml(s.label || 'Untitled')}</strong>
              <span style="font-size:11px;color:var(--muted);margin-left:8px;">${s.id}</span>
              <span style="font-size:11px;color:${s.status === 'complete' ? 'var(--green)' : 'var(--yellow)'};margin-left:8px;">${s.status}</span>
            </div>
            <span style="font-size:11px;color:var(--muted);">${s.created_at ? new Date(s.created_at).toLocaleString() : ''}</span>
          </div>
          <div style="font-size:13px;color:var(--muted);margin-bottom:12px;padding:8px;background:var(--bg);border-radius:4px;">${escapeHtml(truncStr(s.prompt, 200))}</div>
          <div style="display:flex;gap:12px;flex-wrap:wrap;">${resultsHtml}</div>
        </div>
      `;
    }).join('');
  } catch (err) {
    console.error('Compare load error:', err);
  }
}

async function loadCompareAgentList() {
  try {
    const data = await API.get('/status');
    const div = document.getElementById('compare-agent-select');
    const agents = data.agents || [];
    const detected = data.detected || [];

    const allAgents = [...new Set([
      ...agents.map(a => a.name),
      ...detected.map(d => d.name),
    ])];

    if (allAgents.length === 0) {
      div.innerHTML = '<span style="color:var(--muted);font-size:13px;">No agents detected.</span>';
      return;
    }

    div.innerHTML = allAgents.map(name => `
      <label style="display:flex;align-items:center;gap:4px;font-size:13px;cursor:pointer;padding:4px 8px;background:var(--bg);border:1px solid var(--border);border-radius:4px;">
        <input type="checkbox" value="${name}" checked>
        ${name}
      </label>
    `).join('');
  } catch (err) {
    console.error('Compare agent list error:', err);
  }
}

function truncStr(s, max) {
  if (!s || s.length <= max) return s || '';
  return s.substring(0, max) + '...';
}

// ---- Deep Research ----
async function startResearch() {
  const query = document.getElementById('research-query').value.trim();
  const maxDepth = parseInt(document.getElementById('research-depth').value) || 3;

  if (!query) {
    toast('Please enter a research query');
    return;
  }

  try {
    const res = await API.post('/research', { query, max_depth: maxDepth });
    toast(`Research started: ${res.id}`);
    loadResearchSessions();
  } catch (err) {
    toast(`Error: ${err.message}`);
  }
}

async function loadResearchSessions() {
  try {
    const sessions = await API.get('/research');
    const div = document.getElementById('research-sessions');

    if (!sessions || sessions.length === 0) {
      div.innerHTML = '<div class="table-placeholder">No research sessions yet.</div>';
      return;
    }

    div.innerHTML = sessions.map(s => {
      const isComplete = s.status === 'complete';
      const rounds = s.rounds || [];
      const roundsInfo = rounds.map(r =>
        `R${r.number}: ${r.findings?.length || 0} findings, ${r.subtopics?.length || 0} subtopics`
      ).join(' • ');

      return `
        <div class="card" style="margin-bottom:12px;cursor:pointer;" onclick="viewResearch('${s.id}')">
          <div style="display:flex;justify-content:space-between;align-items:center;">
            <div style="flex:1;">
              <div style="display:flex;align-items:center;gap:8px;">
                <strong>${escapeHtml(s.query)}</strong>
                <span class="status-badge" style="font-size:11px;padding:2px 8px;border-radius:10px;background:${isComplete ? 'var(--green)' : 'var(--yellow)'}20;color:${isComplete ? 'var(--green)' : 'var(--yellow)'};">${s.status}</span>
              </div>
              <div style="font-size:12px;color:var(--muted);margin-top:4px;">
                Depth: ${s.depth || 0}/${s.max_depth || 3} • ${roundsInfo}
              </div>
              ${s.progress ? `<div style="font-size:12px;color:var(--accent);margin-top:4px;">${escapeHtml(s.progress)}</div>` : ''}
            </div>
            ${isComplete ? '<span style="color:var(--green);font-size:13px;">📄 View Report →</span>' : '<span style="color:var(--yellow);">⏳</span>'}
          </div>
        </div>
      `;
    }).join('');
  } catch (err) {
    console.error('Research load error:', err);
  }
}

async function viewResearch(id) {
  try {
    const s = await API.get(`/research/${id}`);
    if (!s) return;

    // Build a detailed view as a modal or expandable section
    const div = document.getElementById('research-sessions');
    const findings = s.findings || [];
    const rounds = s.rounds || [];

    let html = `
      <div class="card" style="margin-bottom:12px;border-color:var(--accent);">
        <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:12px;">
          <div>
            <h3 style="margin:0;">${escapeHtml(s.query)}</h3>
            <span style="font-size:12px;color:var(--muted);">${s.id} • Status: ${s.status} • ${s.findings_count || 0} findings • ${rounds.length} rounds</span>
          </div>
          <button class="btn" onclick="loadResearchSessions()">Back</button>
        </div>`;

    // Report
    if (s.report) {
      html += `
        <div style="padding:16px;background:var(--bg);border-radius:6px;margin-bottom:12px;">
          <h4 style="margin:0 0 8px 0;color:var(--accent);">📄 ${escapeHtml(s.report.title)}</h4>
          <p style="font-size:13px;color:var(--fg);">${escapeHtml(s.report.summary)}</p>
          <div style="font-size:12px;color:var(--muted);margin-top:8px;">
            ${s.report.word_count || 0} words • ${s.report.sources?.length || 0} sources
          </div>
          ${s.report.sources?.length > 0 ? `
            <div style="margin-top:8px;">
              <strong style="font-size:13px;">Sources:</strong>
              <ul style="font-size:12px;color:var(--muted);margin:4px 0 0 0;padding-left:20px;">
                ${s.report.sources.map(src => `<li><a href="${escapeHtml(src)}" target="_blank" style="color:var(--accent);">${escapeHtml(src)}</a></li>`).join('')}
              </ul>
            </div>
          ` : ''}
        </div>`;
    }

    // Rounds
    rounds.forEach(r => {
      html += `
        <div style="padding:12px;background:var(--bg-secondary);border-radius:6px;margin-bottom:8px;">
          <div style="display:flex;justify-content:space-between;">
            <strong>Round ${r.number}: ${escapeHtml(r.query)}</strong>
            <span style="font-size:11px;color:var(--muted);">${r.findings?.length || 0} findings</span>
          </div>
          ${r.subtopics?.length > 0 ? `<div style="font-size:12px;color:var(--muted);margin-top:4px;">Subtopics: ${r.subtopics.map(st => escapeHtml(st)).join(', ')}</div>` : ''}
          ${(r.findings || []).map(f => `
            <div style="margin-top:4px;padding:4px 8px;background:var(--bg);border-radius:4px;font-size:12px;">
              <a href="${escapeHtml(f.url)}" target="_blank" style="color:var(--accent);">${escapeHtml(f.title)}</a>
            </div>
          `).join('')}
        </div>`;
    });

    html += '</div>';
    div.innerHTML = html;
  } catch (err) {
    toast(`Error: ${err.message}`);
  }
}

// ---- Search (SearXNG) ----
async function doSearch() {
  const input = document.getElementById('search-input');
  const query = input.value.trim();
  if (!query) return;

  const resultsDiv = document.getElementById('search-results');
  resultsDiv.innerHTML = '<div class="table-placeholder">Searching...</div>';

  try {
    const data = await API.post('/search', { query });
    if (!data.results || data.results.length === 0) {
      resultsDiv.innerHTML = '<div class="table-placeholder">No results found.</div>';
      return;
    }
    resultsDiv.innerHTML = data.results.map(r => `
      <div class="search-result" style="padding:8px 0;border-bottom:1px solid var(--border);">
        <div style="font-weight:500;color:var(--accent);">${escapeHtml(r.title)}</div>
        <div style="font-size:12px;color:var(--muted);">${escapeHtml(r.url)}</div>
        <div style="font-size:13px;margin-top:4px;color:var(--fg);">${escapeHtml(r.content || '')}</div>
      </div>
    `).join('');
  } catch (err) {
    resultsDiv.innerHTML = `<div class="table-placeholder">Error: ${escapeHtml(err.message)}</div>`;
  }
}

// Enter to search
document.addEventListener('DOMContentLoaded', () => {
  const searchInput = document.getElementById('search-input');
  if (searchInput) {
    searchInput.addEventListener('keydown', e => {
      if (e.key === 'Enter') {
        e.preventDefault();
        doSearch();
      }
    });
  }
});

// ---- Memory (ChromaDB) ----
async function loadMemory() {
  try {
    const data = await API.get('/memory/decisions');
    const div = document.getElementById('memory-decisions');
    if (!data.results || data.results.length === 0) {
      div.innerHTML = '<div class="table-placeholder">No decisions stored yet. Start a debate and reach consensus to save one.</div>';
      return;
    }
    div.innerHTML = data.results.map(item => `
      <div class="memory-item" style="padding:8px 0;border-bottom:1px solid var(--border);">
        <div style="font-size:13px;color:var(--fg);">${escapeHtml(item.content)}</div>
        <div style="font-size:11px;color:var(--muted);margin-top:4px;">
          ${item.metadata?.topic ? 'Topic: ' + escapeHtml(item.metadata.topic) : ''}
          ${item.metadata?.consensus ? ' · Consensus: ' + item.metadata.consensus : ''}
          ${item.metadata?.timestamp ? ' · ' + new Date(item.metadata.timestamp).toLocaleString() : ''}
        </div>
      </div>
    `).join('');
  } catch (err) {
    document.getElementById('memory-decisions').innerHTML = `<div class="table-placeholder">Error: ${escapeHtml(err.message)}</div>`;
  }
}

async function queryMemory() {
  const input = document.getElementById('memory-query-input');
  const query = input.value.trim();
  if (!query) return;

  const div = document.getElementById('memory-results');
  div.innerHTML = '<div class="table-placeholder">Querying...</div>';

  try {
    const data = await API.get(`/memory?q=${encodeURIComponent(query)}&collection=decisions`);
    if (!data.results || data.results.length === 0) {
      div.innerHTML = '<div class="table-placeholder">No matching memories found.</div>';
      return;
    }
    div.innerHTML = data.results.map(item => `
      <div class="memory-item" style="padding:8px 0;border-bottom:1px solid var(--border);">
        <div style="font-size:13px;color:var(--fg);">${escapeHtml(item.content)}</div>
        <div style="font-size:11px;color:var(--muted);margin-top:4px;">
          Distance: ${item.distance ? item.distance.toFixed(3) : '-'}
          ${item.metadata?.timestamp ? ' · ' + new Date(item.metadata.timestamp).toLocaleString() : ''}
        </div>
      </div>
    `).join('');
  } catch (err) {
    div.innerHTML = `<div class="table-placeholder">Error: ${escapeHtml(err.message)}</div>`;
  }
}

async function storeMemory() {
  const input = document.getElementById('memory-store-input');
  const content = input.value.trim();
  if (!content) return;

  try {
    const res = await API.post('/memory', {
      collection: 'manual',
      content: content,
      metadata: { source: 'web_ui' }
    });
    toast(`Stored: ${res.id}`);
    input.value = '';
  } catch (err) {
    toast(`Error: ${err.message}`);
  }
}

// ---- Notify (ntfy) ----
async function testNotify() {
  try {
    const res = await API.post('/notify/test', {});
    toast(`✅ ${res.status}`);
  } catch (err) {
    toast(`Error: ${err.message}`);
  }
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

// ---- Polling ----
let pollInterval;

function startPolling() {
  loadDashboard();
  loadAgentsList();
  loadKeys();
  loadConfig();
  loadCompareSessions();
  loadResearchSessions();
  loadMemory();
  checkHealth();
  pollInterval = setInterval(() => {
    loadDashboard();
    loadAgentsList();
    loadDebate();
    loadMemory();
    loadCompareSessions();
    loadResearchSessions();
    checkHealth();
  }, 3000);
}

function stopPolling() {
  if (pollInterval) clearInterval(pollInterval);
}

// ---- Init ----
document.addEventListener('DOMContentLoaded', () => {
  // Auth check
  const meta = document.querySelector('meta[name="amuxasi-auth"]');
  if (meta && meta.getAttribute('content') === 'true') {
    const token = getAuthToken();
    if (!token) promptAuth();
  }

  // Connection status element
  const statusEl = document.querySelector('.nav-status');
  if (statusEl && !document.getElementById('connection-status')) {
    const conn = document.createElement('span');
    conn.id = 'connection-status';
    conn.className = 'status-ok';
    conn.textContent = 'Connected';
    conn.style.cssText = 'font-size:11px;color:var(--muted);margin-left:8px;';
    statusEl.appendChild(conn);
  }

  // Research Enter key
  document.getElementById('research-query')?.addEventListener('keydown', e => {
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); startResearch(); }
  });

  // Search Enter key
  document.getElementById('search-input')?.addEventListener('keydown', e => {
    if (e.key === 'Enter') { e.preventDefault(); doSearch(); }
  });

  // Memory Enter keys
  document.getElementById('memory-query-input')?.addEventListener('keydown', e => {
    if (e.key === 'Enter') { e.preventDefault(); queryMemory(); }
  });
  document.getElementById('memory-store-input')?.addEventListener('keydown', e => {
    if (e.key === 'Enter') { e.preventDefault(); storeMemory(); }
  });

  // Load compare agent list
  loadCompareAgentList();

  // Start everything
  startPolling();

  // Cleanup on page unload
  window.addEventListener('beforeunload', stopPolling);
});
