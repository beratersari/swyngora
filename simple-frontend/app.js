const $ = (id) => document.getElementById(id);

const els = {
  apiBase: $("apiBase"),
  symbol: $("symbol"),
  interval: $("interval"),
  limit: $("limit"),
  status: $("status"),
  ticker: $("ticker"),
  supply: $("supply"),
  candlesBody: $("candlesBody"),
  raw: $("raw"),
  buttons: ["btnAll", "btnCandles", "btnTicker", "btnSupply"].map($),
};

const rawPayload = {};

function setStatus(msg, kind = "") {
  els.status.textContent = msg;
  els.status.className = `status ${kind}`.trim();
}

function setBusy(busy) {
  els.buttons.forEach((b) => {
    b.disabled = busy;
  });
}

function baseUrl() {
  return els.apiBase.value.replace(/\/$/, "");
}

function symbol() {
  return els.symbol.value.trim();
}

function fmtNum(v, digits = 2) {
  if (v === null || v === undefined || v === "") return "—";
  const n = Number(v);
  if (Number.isNaN(n)) return String(v);
  return n.toLocaleString(undefined, { maximumFractionDigits: digits });
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
  const url = `${baseUrl()}${path}`;
  const res = await fetch(url, { headers: { Accept: "application/json" } });
  const text = await res.text();
  let data;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    throw new Error(`Non-JSON response (${res.status}): ${text.slice(0, 200)}`);
  }
  if (!res.ok) {
    const msg = data?.error?.message || res.statusText || "request failed";
    const code = data?.error?.code ? ` [${data.error.code}]` : "";
    throw new Error(`${res.status}${code}: ${msg}`);
  }
  return data;
}

function renderTicker(data) {
  if (!data) {
    els.ticker.innerHTML = `<span class="muted">No data yet.</span>`;
    return;
  }
  els.ticker.innerHTML = `
    <dl>
      <dt>Symbol</dt><dd>${data.symbol}</dd>
      <dt>Last price</dt><dd>${fmtNum(data.lastPrice, 8)}</dd>
      <dt>Change 24h</dt><dd>${fmtNum(data.priceChangePercent, 3)}%</dd>
      <dt>High / Low</dt><dd>${fmtNum(data.highPrice, 8)} / ${fmtNum(data.lowPrice, 8)}</dd>
      <dt>Volume (base)</dt><dd>${fmtNum(data.volume, 4)}</dd>
      <dt>Volume (quote)</dt><dd>${fmtNum(data.quoteVolume, 2)}</dd>
      <dt>Trades</dt><dd>${fmtNum(data.tradeCount, 0)}</dd>
      <dt>Window</dt><dd>${fmtTime(data.openTime)} → ${fmtTime(data.closeTime)}</dd>
      <dt>Exchange</dt><dd>${data.exchange}</dd>
    </dl>
  `;
}

function renderSupply(data) {
  if (!data) {
    els.supply.innerHTML = `<span class="muted">No data yet.</span>`;
    return;
  }
  els.supply.innerHTML = `
    <dl>
      <dt>Asset</dt><dd>${data.asset} <span class="muted">(${data.name || "—"})</span></dd>
      <dt>Circulating</dt><dd>${fmtNum(data.circulatingSupply, 0)}</dd>
      <dt>Total</dt><dd>${fmtNum(data.totalSupply, 0)}</dd>
      <dt>Max</dt><dd>${fmtNum(data.maxSupply, 0)}</dd>
      <dt>Price USD</dt><dd>${fmtNum(data.currentPriceUsd, 4)}</dd>
      <dt>Source</dt><dd>${data.source}</dd>
      <dt>As of</dt><dd>${fmtTime(data.asOf)}</dd>
    </dl>
    <p class="muted" style="margin:0.75rem 0 0;font-size:0.85rem">${data.note || ""}</p>
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
        <td>${fmtTime(c.openTime)}</td>
        <td>${fmtNum(c.open, 8)}</td>
        <td>${fmtNum(c.high, 8)}</td>
        <td>${fmtNum(c.low, 8)}</td>
        <td>${fmtNum(c.close, 8)}</td>
        <td>${fmtNum(c.volume, 4)}</td>
        <td>${fmtNum(c.quoteVolume, 2)}</td>
        <td>${fmtNum(c.tradeCount, 0)}</td>
      </tr>`
    )
    .join("");
}

function updateRaw() {
  els.raw.textContent = JSON.stringify(rawPayload, null, 2);
}

async function fetchCandles() {
  const sym = encodeURIComponent(symbol());
  const interval = encodeURIComponent(els.interval.value);
  const limit = encodeURIComponent(els.limit.value || "24");
  const data = await apiGet(
    `/api/v1/market/candles?symbol=${sym}&interval=${interval}&limit=${limit}`
  );
  rawPayload.candles = data;
  renderCandles(data);
  updateRaw();
  return data;
}

async function fetchTicker() {
  const sym = encodeURIComponent(symbol());
  const data = await apiGet(`/api/v1/market/ticker/24h?symbol=${sym}`);
  rawPayload.ticker = data;
  renderTicker(data);
  updateRaw();
  return data;
}

async function fetchSupply() {
  const sym = encodeURIComponent(symbol());
  const data = await apiGet(`/api/v1/market/supply?symbol=${sym}`);
  rawPayload.supply = data;
  renderSupply(data);
  updateRaw();
  return data;
}

async function run(job, label) {
  setBusy(true);
  setStatus(`${label}…`);
  try {
    await job();
    setStatus(`${label} — ok`, "ok");
  } catch (err) {
    console.error(err);
    setStatus(String(err.message || err), "error");
  } finally {
    setBusy(false);
  }
}

$("btnCandles").addEventListener("click", () => run(fetchCandles, "Candles"));
$("btnTicker").addEventListener("click", () => run(fetchTicker, "24h ticker"));
$("btnSupply").addEventListener("click", () => run(fetchSupply, "Supply"));
$("btnAll").addEventListener("click", () =>
  run(async () => {
    await Promise.all([fetchCandles(), fetchTicker(), fetchSupply()]);
  }, "Fetch all")
);

// Load intervals list if API is up (non-fatal).
(async () => {
  try {
    const data = await apiGet("/api/v1/market/intervals");
    if (Array.isArray(data.intervals) && data.intervals.length) {
      const sel = els.interval;
      const current = sel.value;
      sel.innerHTML = data.intervals
        .map((iv) => `<option value="${iv}">${iv}</option>`)
        .join("");
      sel.value = data.intervals.includes(current) ? current : "1h";
    }
  } catch {
    /* backend may not be running yet */
  }
})();
