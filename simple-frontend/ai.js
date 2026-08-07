/**
 * Simple-frontend AI chat harness → POST /api/v1/ai/chat
 * Multi-turn via sessionId stored in localStorage.
 */
const $ = (id) => document.getElementById(id);
const API_BASE_KEY = "swyngora.simple-frontend.apiBase.v1";
const SESSION_KEY = "swyngora.simple-frontend.aiSession.v1";
const MAX_LEN = 4000;

const SUGGESTIONS = [
  "What is BTC RSI on binance 1h?",
  "What is the ETH price on binance?",
  "Any recent pumps on binance USDT pairs?",
];

const els = {
  apiBase: $("apiBase"),
  status: $("status"),
  thread: $("thread"),
  message: $("message"),
  chatForm: $("chatForm"),
  btnSend: $("btnSend"),
  btnNewChat: $("btnNewChat"),
  sessionIdLabel: $("sessionIdLabel"),
  suggestions: $("suggestions"),
};

let sessionId = loadSessionId();
let pending = false;

function loadApiBase() {
  try {
    const v = localStorage.getItem(API_BASE_KEY);
    if (v) return v;
  } catch { /* ignore */ }
  return "http://localhost:8080";
}

function saveApiBase() {
  try {
    localStorage.setItem(API_BASE_KEY, baseUrl());
  } catch { /* ignore */ }
}

function baseUrl() {
  return (els.apiBase.value || "").replace(/\/$/, "");
}

function randomId() {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID().slice(0, 8);
  }
  return Math.random().toString(36).slice(2, 10);
}

function loadSessionId() {
  try {
    const v = localStorage.getItem(SESSION_KEY);
    if (v && v.trim()) return v.trim();
  } catch { /* ignore */ }
  return newSessionId();
}

function newSessionId() {
  const id = `simple-ai-${randomId()}`;
  try {
    localStorage.setItem(SESSION_KEY, id);
  } catch { /* ignore */ }
  return id;
}

function setStatus(msg, kind = "") {
  els.status.textContent = msg;
  els.status.className = `status ${kind}`.trim();
}

function escapeHtml(s) {
  if (globalThis.SwyngoraMarkdown?.escapeHtml) {
    return globalThis.SwyngoraMarkdown.escapeHtml(s);
  }
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

/** Render assistant Markdown; user/error stay plain (escaped) for safety/clarity. */
function formatBubbleHtml(role, content, isError) {
  const text = String(content ?? "");
  if (role === "assistant" && !isError && globalThis.SwyngoraMarkdown?.render) {
    return globalThis.SwyngoraMarkdown.render(text);
  }
  return escapeHtml(text).replace(/\n/g, "<br>");
}

function mdInline(text) {
  if (globalThis.SwyngoraMarkdown?.renderInline) {
    return globalThis.SwyngoraMarkdown.renderInline(text);
  }
  return escapeHtml(text);
}

function mdBlock(text) {
  if (globalThis.SwyngoraMarkdown?.render) {
    return globalThis.SwyngoraMarkdown.render(text);
  }
  return escapeHtml(text).replace(/\n/g, "<br>");
}

/** Short label for tool chips (full text goes in expandable detail). */
function shortToolLabel(tool) {
  const s = String(tool || "").trim();
  if (s.length <= 48) return s;
  // Prefer name before "(" or "→"
  const m = s.match(/^([a-zA-Z0-9_./-]+)/);
  if (m && m[1].length >= 3) return m[1];
  return `${s.slice(0, 45)}…`;
}

function appendProcessDetails(bubble, thinking, tools) {
  const hasThinking = Array.isArray(thinking) && thinking.length > 0;
  const hasTools = Array.isArray(tools) && tools.length > 0;
  if (!hasThinking && !hasTools) return;

  const details = document.createElement("details");
  details.className = "ai-process";
  const summary = document.createElement("summary");
  summary.className = "ai-process-summary";
  const parts = [];
  if (hasThinking) parts.push(`${thinking.length} step${thinking.length === 1 ? "" : "s"}`);
  if (hasTools) parts.push(`${tools.length} tool${tools.length === 1 ? "" : "s"}`);
  summary.textContent = `Process · ${parts.join(" · ")}`;
  details.appendChild(summary);

  if (hasThinking) {
    const ol = document.createElement("ol");
    ol.className = "ai-process-list";
    thinking.forEach((line) => {
      const li = document.createElement("li");
      li.className = "ai-process-item md-body";
      // Long agent lines may contain Markdown (**bold**, etc.)
      li.innerHTML = mdBlock(String(line));
      ol.appendChild(li);
    });
    details.appendChild(ol);
  }

  if (hasTools) {
    const wrap = document.createElement("div");
    wrap.className = "ai-tools-block";
    const title = document.createElement("div");
    title.className = "ai-process-subtitle";
    title.textContent = "Tools";
    wrap.appendChild(title);

    const chips = document.createElement("div");
    chips.className = "ai-meta";
    tools.forEach((tool) => {
      const chip = document.createElement("span");
      chip.className = "ai-chip tool";
      chip.title = String(tool);
      chip.innerHTML = mdInline(shortToolLabel(tool));
      chips.appendChild(chip);
    });
    wrap.appendChild(chips);

    // Full tool lines (often include Markdown + JSON snippets)
    const full = document.createElement("div");
    full.className = "ai-tools-full md-body";
    full.innerHTML = mdBlock(
      tools.map((t) => `- ${String(t).replace(/\n/g, " ")}`).join("\n")
    );
    wrap.appendChild(full);
    details.appendChild(wrap);
  }

  bubble.appendChild(details);
}

function updateSessionLabel() {
  if (els.sessionIdLabel) els.sessionIdLabel.textContent = sessionId;
}

function appendBubble(role, content, opts = {}) {
  const row = document.createElement("div");
  row.className = `ai-bubble-row ${role}`;
  const bubble = document.createElement("div");
  bubble.className = `ai-bubble ${role}${opts.isError ? " error" : ""}`;

  const who = document.createElement("div");
  who.className = "ai-bubble-who";
  who.textContent = role === "user" ? "You" : "Assistant";
  bubble.appendChild(who);

  const body = document.createElement("div");
  body.className = `ai-bubble-body${role === "assistant" && !opts.isError ? " md-body" : ""}`;
  body.innerHTML = formatBubbleHtml(role, content, opts.isError);
  bubble.appendChild(body);

  if (role === "assistant" && !opts.isError) {
    appendProcessDetails(bubble, opts.thinking, opts.tools);
  }

  row.appendChild(bubble);
  els.thread.appendChild(row);
  els.thread.scrollTop = els.thread.scrollHeight;
  return bubble;
}

function setPending(on) {
  pending = on;
  els.btnSend.disabled = on;
  els.message.disabled = on;
  els.btnNewChat.disabled = on;
}

function showEmptyHint() {
  if (els.thread.childElementCount > 0) return;
  const hint = document.createElement("div");
  hint.className = "ai-empty muted";
  hint.id = "emptyHint";
  hint.textContent = "Start a conversation, or pick a suggestion below.";
  els.thread.appendChild(hint);
}

function clearEmptyHint() {
  const h = $("emptyHint");
  if (h) h.remove();
}

async function sendMessage(raw) {
  const text = String(raw || "").trim().slice(0, MAX_LEN);
  if (!text || pending) return;

  clearEmptyHint();
  els.message.value = "";
  appendBubble("user", text);
  setPending(true);
  setStatus("Thinking… (multi-agent may take up to a few minutes)", "");

  const thinkingBubble = appendBubble("assistant", "…");

  try {
    saveApiBase();
    const res = await fetch(`${baseUrl()}/api/v1/ai/chat`, {
      method: "POST",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ message: text, sessionId }),
    });
    const data = await res.json().catch(() => ({}));
    thinkingBubble.parentElement?.remove();

    if (!res.ok) {
      const code = data?.error?.code || res.status;
      let msg = data?.error?.message || `Request failed (${code})`;
      // Backend maps all upstream failures to a generic string — add setup hints for chat.
      if (res.status === 502 || res.status === 503 || /upstream/i.test(msg)) {
        msg =
          "AI upstream unavailable. Need: (1) backend :8080, (2) Python AI :8090, " +
          "(3) LLM — Ollama on :11434 or XAI_API_KEY with AI_LLM_PROVIDER=grok. " +
          "From ai/:  export SWYNGORA_API_URL=http://127.0.0.1:8080 AI_LLM_PROVIDER=grok XAI_API_KEY=…  " +
          "&& .venv/bin/python -m swyngora_ai.serve --host 127.0.0.1 --port 8090  " +
          `(API said: ${data?.error?.message || res.status})`;
      }
      appendBubble("assistant", msg, { isError: true });
      setStatus(msg, "error");
      return;
    }

    appendBubble("assistant", data.reply || "—", {
      tools: Array.isArray(data.tools) ? data.tools : [],
      thinking: Array.isArray(data.thinking) ? data.thinking : [],
    });
    if (data.sessionId) {
      sessionId = String(data.sessionId);
      try {
        localStorage.setItem(SESSION_KEY, sessionId);
      } catch { /* ignore */ }
      updateSessionLabel();
    }
    setStatus(data.note || "Done.", "ok");
  } catch (err) {
    thinkingBubble.parentElement?.remove();
    const msg = `Cannot reach API at ${baseUrl()} — is the backend running? ${err?.message || err}`;
    appendBubble("assistant", msg, { isError: true });
    setStatus(msg, "error");
  } finally {
    setPending(false);
  }
}

function renderSuggestions() {
  els.suggestions.innerHTML = "";
  SUGGESTIONS.forEach((text) => {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "ghost small";
    btn.textContent = text;
    btn.addEventListener("click", () => {
      void sendMessage(text);
    });
    els.suggestions.appendChild(btn);
  });
}

function onNewChat() {
  if (pending) return;
  els.thread.innerHTML = "";
  sessionId = newSessionId();
  updateSessionLabel();
  showEmptyHint();
  setStatus("New session started.", "ok");
  els.message.focus();
}

function init() {
  els.apiBase.value = loadApiBase();
  updateSessionLabel();
  renderSuggestions();
  showEmptyHint();

  els.apiBase.addEventListener("change", saveApiBase);
  els.btnNewChat.addEventListener("click", onNewChat);
  els.chatForm.addEventListener("submit", (e) => {
    e.preventDefault();
    void sendMessage(els.message.value);
  });
  els.message.addEventListener("keydown", (e) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      void sendMessage(els.message.value);
    }
  });

  // Prefill from ?q= or ?message=
  const params = new URLSearchParams(location.search);
  const pre = params.get("q") || params.get("message");
  if (pre) {
    els.message.value = pre;
  }
}

init();
