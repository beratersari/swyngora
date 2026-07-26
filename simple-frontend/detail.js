const $ = (id) => document.getElementById(id);
const API_BASE_KEY = "swyngora.simple-frontend.apiBase.v1";
const WATCHLIST_STORAGE_KEY = "swyngora.simple-frontend.watchlist.v1";
const CLIENT_ID_KEY = "swyngora.simple-frontend.clientId.v1";

const params = new URLSearchParams(location.search);

const els = {
  apiBase: $("apiBase"),
  exchange: $("exchange"),
  symbol: $("symbol"),
  interval: $("interval"),
  limit: $("limit"),
  status: $("status"),
  ticker: $("ticker"),
  supply: $("supply"),
  indicators: $("indicators"),
  candlesBody: $("candlesBody"),
  pageTitle: $("pageTitle"),
  pageSub: $("pageSub"),
  btnReload: $("btnReload"),
  btnWatch: $("btnWatch"),
};

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

function setStatus(msg, kind = "") {
  els.status.textContent = msg;
  els.status.className = `status ${kind}`.trim();
}

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

/** Format prices without collapsing tiny non-zero values to "0". */
function fmtNum(v, digits = 2) {
  if (typeof globalThis.WatchlistLogic?.fmtNum === "function") {
    return escapeHtml(globalThis.WatchlistLogic.fmtNum(v, digits));
  }
  if (v === null || v === undefined || v === "") return "—";
  const n = Number(v);
  if (Number.isNaN(n)) return escapeHtml(String(v));
  if (n === 0) return escapeHtml("0");
  const abs = Math.abs(n);
  if (abs > 0 && abs < Math.pow(10, -digits)) {
    return escapeHtml(n.toExponential(Math.min(4, digits)));
  }
  return escapeHtml(
    n.toLocaleString(undefined, {
      maximumFractionDigits: digits,
      minimumFractionDigits: 0,
    })
  );
}

/** Current venue from the detail form (binance | coinbase | bybit). */
function currentExchange() {
  return String(els.exchange?.value || "binance").toLowerCase();
}

/**
 * 24h / per-bar trade counts are only published by Binance public APIs.
 * Bybit and Coinbase always send 0 — show em dash so detail matches the dashboard.
 */
function fmtTradeCount(v, exchange) {
  const ex = String(exchange || currentExchange()).toLowerCase();
  const n = Number(v);
  if (ex === "bybit" || ex === "coinbase") {
    if (!Number.isFinite(n) || n === 0) return "—";
  }
  if (v === null || v === undefined || v === "") return "—";
  return fmtNum(v, 0);
}

/** High/low pair; empty strings render as em dash (e.g. before Coinbase stats fill). */
function fmtRange(high, low) {
  const h = high === null || high === undefined || high === "" ? "—" : fmtNum(high, 8);
  const l = low === null || low === undefined || low === "" ? "—" : fmtNum(low, 8);
  return `${h} / ${l}`;
}

/** Monotonic load generation — ignore stale responses after rapid symbol/exchange changes. */
let detailSeq = 0;

function fmtChange(v) {
  if (v === null || v === undefined || v === "") return "—";
  const n = Number(v);
  if (Number.isNaN(n)) return escapeHtml(String(v));
  const s = n.toLocaleString(undefined, { maximumFractionDigits: 3 });
  return escapeHtml(`${n > 0 ? "+" : ""}${s}%`);
}

function changeClass(v) {
  const n = Number(v);
  if (Number.isNaN(n) || n === 0) return "";
  return n > 0 ? "pos" : "neg";
}

function fmtTime(iso) {
  if (!iso) return "—";
  try {
    return new Date(iso).toLocaleString();
  } catch {
    return iso;
  }
}

async function apiGet(path) {
  const res = await fetch(`${baseUrl()}${path}`, {
    headers: { Accept: "application/json" },
  });
  const text = await res.text();
  let data;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    throw new Error(`Non-JSON (${res.status})`);
  }
  if (!res.ok) {
    throw new Error(data?.error?.message || res.statusText || "request failed");
  }
  return data;
}

async function apiSend(method, path, body) {
  const res = await fetch(`${baseUrl()}${path}`, {
    method,
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      "X-Client-Id": clientId(),
    },
    body: body != null ? JSON.stringify(body) : undefined,
  });
  const text = await res.text();
  let data = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    throw new Error(`Non-JSON (${res.status})`);
  }
  if (!res.ok) throw new Error(data?.error?.message || res.statusText || "failed");
  return data;
}

function clientId() {
  try {
    let id = localStorage.getItem(CLIENT_ID_KEY);
    if (!id) {
      id = "web-" + Math.random().toString(36).slice(2, 10) + Date.now().toString(36);
      localStorage.setItem(CLIENT_ID_KEY, id);
    }
    return id;
  } catch {
    if (!globalThis.__swyngoraClientId) {
      globalThis.__swyngoraClientId =
        "web-" + Math.random().toString(36).slice(2, 10) + Date.now().toString(36);
    }
    return globalThis.__swyngoraClientId;
  }
}

function loadWatchlistLocal() {
  try {
    const raw = localStorage.getItem(WATCHLIST_STORAGE_KEY);
    if (!raw) return [];
    const arr = JSON.parse(raw);
    return Array.isArray(arr) ? arr : [];
  } catch {
    return [];
  }
}

function saveWatchlistLocal(list) {
  localStorage.setItem(WATCHLIST_STORAGE_KEY, JSON.stringify(list));
}

function isWatched(exchange, symbol) {
  const ex = String(exchange || "binance").toLowerCase();
  const sym = String(symbol || "").toUpperCase();
  return loadWatchlistLocal().some(
    (w) =>
      String(w.exchange || "binance").toLowerCase() === ex &&
      String(w.symbol || "").toUpperCase() === sym
  );
}

function paintWatchBtn() {
  const on = isWatched(els.exchange.value, els.symbol.value);
  els.btnWatch.textContent = on ? "★ Watching" : "☆ Watch";
  els.btnWatch.classList.toggle("on", on);
}

function toggleWatch() {
  const exchange = String(els.exchange.value || "binance").toLowerCase();
  const symbol = String(els.symbol.value || "").toUpperCase();
  if (!symbol) return;
  let list = loadWatchlistLocal();
  const key = `${exchange}|${symbol}`;
  const has = list.some(
    (w) => `${String(w.exchange || "binance").toLowerCase()}|${String(w.symbol || "").toUpperCase()}` === key
  );
  if (has) {
    list = list.filter(
      (w) =>
        `${String(w.exchange || "binance").toLowerCase()}|${String(w.symbol || "").toUpperCase()}` !== key
    );
    saveWatchlistLocal(list);
    paintWatchBtn();
    apiSend(
      "DELETE",
      `/api/v1/watchlist/items?clientId=${encodeURIComponent(clientId())}&exchange=${encodeURIComponent(exchange)}&symbol=${encodeURIComponent(symbol)}`
    ).catch(() => {});
  } else {
    list.push({ exchange, symbol });
    saveWatchlistLocal(list);
    paintWatchBtn();
    apiSend("POST", "/api/v1/watchlist/items", {
      clientId: clientId(),
      exchange,
      symbol,
    }).catch(() => {});
  }
}

function renderTicker(data) {
  if (!data) {
    els.ticker.innerHTML = `<span class="muted">—</span>`;
    return;
  }
  els.ticker.innerHTML = `
    <dl>
      <dt>Last</dt><dd>${fmtNum(data.lastPrice, 8)}</dd>
      <dt>Change 24h</dt><dd class="${changeClass(data.priceChangePercent)}">${fmtChange(data.priceChangePercent)}</dd>
      <dt>High / Low</dt><dd>${fmtRange(data.highPrice, data.lowPrice)}</dd>
      <dt>Volume</dt><dd>${fmtNum(data.volume, 4)}</dd>
      <dt>Quote vol</dt><dd>${fmtNum(data.quoteVolume, 2)}</dd>
      <dt>Trades</dt><dd>${fmtTradeCount(data.tradeCount, data.exchange || currentExchange())}</dd>
    </dl>
  `;
}

function renderSupply(data) {
  if (!data || data._error) {
    els.supply.innerHTML = `<span class="muted">${escapeHtml(data?._error || "—")}</span>`;
    return;
  }
  const asOf = data.asOf ? fmtTime(data.asOf) : "—";
  els.supply.innerHTML = `
    <dl>
      <dt>Asset</dt><dd>${escapeHtml(data.asset)} <span class="muted">(${escapeHtml(data.name || "—")})</span></dd>
      <dt>Circulating</dt><dd>${fmtNum(data.circulatingSupply, 0)}</dd>
      <dt>Total</dt><dd>${fmtNum(data.totalSupply, 0)}</dd>
      <dt>Max</dt><dd>${fmtNum(data.maxSupply, 0)}</dd>
      <dt>Price USD</dt><dd>${fmtNum(data.currentPriceUsd, 4)}</dd>
      <dt>As of</dt><dd>${escapeHtml(asOf)} <span class="muted">(daily snapshot)</span></dd>
      <dt>Source</dt><dd>${escapeHtml(data.source || "—")}</dd>
    </dl>
  `;
}

function renderIndicators(data) {
  if (!data || data._error) {
    els.indicators.innerHTML = `<span class="muted">${escapeHtml(data?._error || "—")}</span>`;
    return;
  }
  const rsi = data.latest?.rsi;
  const ema = data.latest?.ema || {};
  let rsiClass = "";
  if (rsi != null) {
    if (rsi >= 70) rsiClass = "rsi-over";
    else if (rsi <= 30) rsiClass = "rsi-under";
  }
  const ema12 = ema["12"] ?? ema[12];
  const ema26 = ema["26"] ?? ema[26];
  els.indicators.innerHTML = `
    <dl class="ind-grid">
      <div><dt>RSI (14)</dt><dd class="${rsiClass}">${rsi == null ? "—" : fmtNum(rsi, 2)}</dd></div>
      <div><dt>EMA 12</dt><dd>${ema12 == null ? "—" : fmtNum(ema12, 6)}</dd></div>
      <div><dt>EMA 26</dt><dd>${ema26 == null ? "—" : fmtNum(ema26, 6)}</dd></div>
      <div><dt>Interval</dt><dd>${escapeHtml(data.interval || "—")}</dd></div>
    </dl>
  `;
}

function renderCandles(data) {
  const rows = data?.candles || [];
  if (!rows.length) {
    els.candlesBody.innerHTML = `<tr><td colspan="8" class="muted">No candles.</td></tr>`;
    return;
  }
  els.candlesBody.innerHTML = rows
    .slice()
    .reverse()
    .map(
      (c) => `
      <tr>
        <td class="td-left">${fmtTime(c.openTime)}</td>
        <td>${fmtNum(c.open, 8)}</td>
        <td>${fmtNum(c.high, 8)}</td>
        <td>${fmtNum(c.low, 8)}</td>
        <td>${fmtNum(c.close, 8)}</td>
        <td>${fmtNum(c.volume, 4)}</td>
        <td>${fmtNum(c.quoteVolume, 2)}</td>
        <td>${fmtTradeCount(c.tradeCount)}</td>
      </tr>`
    )
    .join("");
}

function syncUrl() {
  const q = new URLSearchParams({
    symbol: els.symbol.value.trim().toUpperCase(),
    exchange: els.exchange.value,
    interval: els.interval.value,
    limit: els.limit.value || "48",
  });
  history.replaceState(null, "", `detail.html?${q.toString()}`);
  document.title = `Swyngora · ${els.symbol.value || "detail"}`;
  els.pageTitle.textContent = els.symbol.value || "Coin detail";
  const qHint = els.exchange.value === "coinbase" ? "USD quote markets" : "USDT quote markets";
  els.pageSub.textContent = `${els.exchange.value} · ${els.interval.value} · ${qHint}`;
}

async function loadDetail() {
  const seq = ++detailSeq;
  const exchange = els.exchange.value || "binance";
  const sym = (els.symbol.value || "").trim().toUpperCase();
  if (!sym) {
    setStatus("Symbol is required", "error");
    return;
  }
  els.symbol.value = sym;
  saveApiBase();
  syncUrl();
  paintWatchBtn();
  setStatus(`Loading ${sym}…`);
  const lim = els.limit.value || "48";
  const indLimit = Math.max(Number(lim) || 48, 60);
  try {
    // Load venue intervals once
    try {
      const ivs = await apiGet(`/api/v1/market/intervals?exchange=${encodeURIComponent(exchange)}`);
      if (seq !== detailSeq) return;
      if (Array.isArray(ivs.intervals) && ivs.intervals.length) {
        const cur = els.interval.value;
        els.interval.innerHTML = ivs.intervals
          .map((x) => `<option value="${escapeHtml(x)}">${escapeHtml(x)}</option>`)
          .join("");
        els.interval.value = ivs.intervals.includes(cur) ? cur : ivs.intervals.includes("1h") ? "1h" : ivs.intervals[0];
      }
    } catch { /* ignore */ }

    if (seq !== detailSeq) return;

    const [ticker, supply, candles, indicators] = await Promise.all([
      apiGet(
        `/api/v1/market/ticker/24h?exchange=${encodeURIComponent(exchange)}&symbol=${encodeURIComponent(sym)}`
      ),
      apiGet(`/api/v1/market/supply?symbol=${encodeURIComponent(sym)}`).catch((e) => ({
        _error: String(e.message || e),
      })),
      apiGet(
        `/api/v1/market/candles?exchange=${encodeURIComponent(exchange)}&symbol=${encodeURIComponent(sym)}&interval=${encodeURIComponent(els.interval.value)}&limit=${encodeURIComponent(lim)}`
      ),
      apiGet(
        `/api/v1/market/indicators?exchange=${encodeURIComponent(exchange)}&symbol=${encodeURIComponent(sym)}&interval=${encodeURIComponent(els.interval.value)}&limit=${encodeURIComponent(indLimit)}&rsiPeriod=14&emaPeriods=12,26`
      ).catch((e) => ({ _error: String(e.message || e) })),
    ]);
    if (seq !== detailSeq) return;
    renderTicker(ticker);
    renderSupply(supply);
    renderIndicators(indicators);
    renderCandles(candles);
    setStatus(`Ready · ${sym} · ${exchange}`, "ok");
  } catch (err) {
    if (seq !== detailSeq) return;
    console.error(err);
    setStatus(String(err.message || err), "error");
  }
}

// Init from query
els.apiBase.value = loadApiBase();
els.exchange.value = (params.get("exchange") || "binance").toLowerCase();
els.symbol.value = (params.get("symbol") || "").toUpperCase();
if (params.get("interval")) els.interval.value = params.get("interval");
if (params.get("limit")) els.limit.value = params.get("limit");

els.btnReload.addEventListener("click", () => loadDetail());
els.btnWatch.addEventListener("click", () => toggleWatch());
els.apiBase.addEventListener("change", () => {
  saveApiBase();
  loadDetail();
});
els.exchange.addEventListener("change", () => loadDetail());
els.interval.addEventListener("change", () => loadDetail());
els.limit.addEventListener("change", () => loadDetail());
els.symbol.addEventListener("keydown", (e) => {
  if (e.key === "Enter") loadDetail();
});

if (els.symbol.value) {
  loadDetail();
} else {
  setStatus("Add ?symbol=BTCUSDT to the URL, or type a symbol and click Reload.", "error");
  els.pageSub.textContent = "No symbol selected";
}
