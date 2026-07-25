const $ = (id) => document.getElementById(id);

const STORAGE_KEY = "swyngora.simple-frontend.columns.v2";
const SORT_STORAGE_KEY = "swyngora.simple-frontend.sort.v1";

/** @typedef {{ id: string, label: string, sortKey: string|null, defaultVisible: boolean, align?: 'left'|'right', format: (row: object) => string }} ColumnDef */

/** @type {ColumnDef[]} */
const COLUMN_DEFS = [
  { id: "symbol", label: "Symbol", sortKey: "symbol", defaultVisible: true, align: "left", format: (m) => m.symbol || "—" },
  {
    id: "tags",
    label: "Tags",
    sortKey: "tags",
    defaultVisible: true,
    align: "left",
    format: (m) => formatTags(m.tags),
  },
  { id: "lastPrice", label: "Last", sortKey: "lastPrice", defaultVisible: true, format: (m) => fmtNum(m.lastPrice, 8) },
  { id: "priceChangePercent", label: "Change %", sortKey: "priceChangePercent", defaultVisible: true, format: (m) => fmtChange(m.priceChangePercent) },
  { id: "volume", label: "Volume", sortKey: "volume", defaultVisible: true, format: (m) => fmtNum(m.volume, 4) },
  { id: "quoteVolume", label: "Quote vol", sortKey: "quoteVolume", defaultVisible: true, format: (m) => fmtNum(m.quoteVolume, 2) },
  { id: "tradeCount", label: "Trades", sortKey: "tradeCount", defaultVisible: false, format: (m) => fmtNum(m.tradeCount, 0) },
  { id: "circulatingSupply", label: "Circ. supply", sortKey: null, defaultVisible: false, format: (m) => fmtNum(m.circulatingSupply, 0) },
  { id: "totalSupply", label: "Total supply", sortKey: null, defaultVisible: false, format: (m) => fmtNum(m.totalSupply, 0) },
  { id: "maxSupply", label: "Max supply", sortKey: null, defaultVisible: false, format: (m) => fmtNum(m.maxSupply, 0) },
  { id: "marketCapCirculating", label: "Mcap (circ)", sortKey: "marketCapCirculating", defaultVisible: true, format: (m) => fmtMcap(m.marketCapCirculating) },
  { id: "marketCapTotal", label: "Mcap (total)", sortKey: "marketCapTotal", defaultVisible: true, format: (m) => fmtMcap(m.marketCapTotal) },
  { id: "marketCapMax", label: "Mcap (max)", sortKey: "marketCapMax", defaultVisible: true, format: (m) => fmtMcap(m.marketCapMax) },
];

const COL_BY_ID = Object.fromEntries(COLUMN_DEFS.map((c) => [c.id, c]));
const ALL_IDS = COLUMN_DEFS.map((c) => c.id);

const LIVE_STORAGE_KEY = "swyngora.simple-frontend.liveInterval.v1";
/** Fields that flash green/red when the numeric value moves between polls. */
const ANIMATED_FIELDS = new Set([
  "lastPrice",
  "priceChangePercent",
  "volume",
  "quoteVolume",
  "tradeCount",
  "marketCapCirculating",
  "marketCapTotal",
  "marketCapMax",
]);

const els = {
  apiBase: $("apiBase"),
  status: $("status"),
  spotQ: $("spotQ"),
  spotExchange: $("spotExchange"),
  spotQuote: $("spotQuote"),
  spotTag: $("spotTag"),
  spotLimit: $("spotLimit"),
  liveInterval: $("liveInterval"),
  liveBadge: $("liveBadge"),
  spotHead: $("spotHead"),
  spotBody: $("spotBody"),
  spotMeta: $("spotMeta"),
  btnRefresh: $("btnRefresh"),
  columnChips: $("columnChips"),
  columnSummary: $("columnSummary"),
  btnColumnsAll: $("btnColumnsAll"),
  btnColumnsDefaults: $("btnColumnsDefaults"),
  btnColumnsMinimal: $("btnColumnsMinimal"),
  detailSection: $("detailSection"),
  detailSymbol: $("detailSymbol"),
  selectedHint: $("selectedHint"),
  interval: $("interval"),
  limit: $("limit"),
  ticker: $("ticker"),
  supply: $("supply"),
  candlesBody: $("candlesBody"),
  btnDetailRefresh: $("btnDetailRefresh"),
  btnDetailClose: $("btnDetailClose"),
};

/**
 * Column layout: ordered list of visible column ids.
 * Symbol is always first and always present.
 * @type {string[]}
 */
let columnOrder = loadColumnOrder();

/** @type {{ sort: string, order: 'asc'|'desc' }} */
let sortState = loadSortState();

/** @type {string|null} */
let selectedSymbol = null;

/** @type {object|null} */
let lastSpotData = null;

/** Fingerprint of last prices — status can say "polled" vs "prices moved". */
let lastSpotFingerprint = "";

/** @type {Map<string, Record<string, number|string|null|undefined>>} previous row snapshots for flash anims */
let prevRowSnapshot = new Map();

function spotFingerprint(data) {
  if (!data || !Array.isArray(data.items)) return "";
  // Only price/volume fields so column layout changes do not count as a market move.
  return data.items
    .map((m) => `${m.symbol}:${m.lastPrice}:${m.quoteVolume}:${m.priceChangePercent}`)
    .join("|");
}

let searchDebounce = null;
let loadSeq = 0;
let dragFromId = null;
let pollTimer = null;
let liveSeconds = loadLiveSeconds();

function loadLiveSeconds() {
  try {
    const v = localStorage.getItem(LIVE_STORAGE_KEY);
    if (v === null) return 10;
    const n = Number(v);
    if ([0, 5, 10, 15, 30].includes(n)) return n;
  } catch {
    /* ignore */
  }
  return 10;
}

function saveLiveSeconds() {
  localStorage.setItem(LIVE_STORAGE_KEY, String(liveSeconds));
}

function toNum(v) {
  if (v === null || v === undefined || v === "" || v === "∞" || v === "Infinity") return null;
  const n = Number(v);
  return Number.isFinite(n) ? n : null;
}

/** @returns {'up'|'down'|''} */
function priceMove(prev, next) {
  const a = toNum(prev);
  const b = toNum(next);
  if (a === null || b === null) return "";
  if (b > a) return "up";
  if (b < a) return "down";
  return "";
}

function updateLiveBadge() {
  if (!els.liveBadge) return;
  if (liveSeconds > 0 && !document.hidden) {
    els.liveBadge.textContent = `LIVE · ${liveSeconds}s`;
    els.liveBadge.className = "live-badge on";
    els.liveBadge.title = `Auto-refresh every ${liveSeconds}s (pauses when tab is hidden). Backend spot cache ~10s.`;
  } else if (liveSeconds > 0 && document.hidden) {
    els.liveBadge.textContent = "LIVE paused";
    els.liveBadge.className = "live-badge paused";
    els.liveBadge.title = "Tab hidden — polling paused";
  } else {
    els.liveBadge.textContent = "LIVE off";
    els.liveBadge.className = "live-badge off";
    els.liveBadge.title = "Auto-refresh off — use Refresh now";
  }
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer);
    pollTimer = null;
  }
}

function startPolling() {
  stopPolling();
  updateLiveBadge();
  if (liveSeconds <= 0) return;
  pollTimer = setInterval(() => {
    if (document.hidden) return;
    loadMarkets({ silent: true });
  }, liveSeconds * 1000);
}

function defaultOrder() {
  return COLUMN_DEFS.filter((c) => c.defaultVisible).map((c) => c.id);
}

function minimalOrder() {
  return ["symbol", "lastPrice", "priceChangePercent", "quoteVolume", "marketCapCirculating"];
}

function normalizeOrder(ids) {
  const known = new Set(ALL_IDS);
  const seen = new Set();
  const out = [];
  for (const id of ids) {
    if (!known.has(id) || seen.has(id)) continue;
    seen.add(id);
    out.push(id);
  }
  if (!out.includes("symbol")) out.unshift("symbol");
  else {
    // Keep symbol first for a stable dashboard
    const rest = out.filter((id) => id !== "symbol");
    return ["symbol", ...rest];
  }
  return out.length ? out : defaultOrder();
}

function loadColumnOrder() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return defaultOrder();
    const parsed = JSON.parse(raw);
    if (Array.isArray(parsed)) return normalizeOrder(parsed);
    // migrate v1 shape { visible: string[] }
    if (parsed && Array.isArray(parsed.visible)) return normalizeOrder(parsed.visible);
    return defaultOrder();
  } catch {
    return defaultOrder();
  }
}

function saveColumnOrder() {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(columnOrder));
}

function loadSortState() {
  try {
    const raw = localStorage.getItem(SORT_STORAGE_KEY);
    if (!raw) return { sort: "quoteVolume", order: "desc" };
    const s = JSON.parse(raw);
    const ok = COLUMN_DEFS.some((c) => c.sortKey === s.sort);
    return {
      sort: ok ? s.sort : "quoteVolume",
      order: s.order === "asc" ? "asc" : "desc",
    };
  } catch {
    return { sort: "quoteVolume", order: "desc" };
  }
}

function saveSortState() {
  localStorage.setItem(SORT_STORAGE_KEY, JSON.stringify(sortState));
}

function isVisible(id) {
  return columnOrder.includes(id);
}

function setStatus(msg, kind = "") {
  els.status.textContent = msg;
  els.status.className = `status ${kind}`.trim();
}

function baseUrl() {
  return els.apiBase.value.replace(/\/$/, "");
}

function fmtNum(v, digits = 2) {
  if (v === null || v === undefined || v === "") return "—";
  const n = Number(v);
  if (Number.isNaN(n)) return escapeHtml(String(v));
  return escapeHtml(n.toLocaleString(undefined, { maximumFractionDigits: digits }));
}

function fmtChange(v) {
  if (v === null || v === undefined || v === "") return "—";
  const n = Number(v);
  if (Number.isNaN(n)) return escapeHtml(String(v));
  const s = n.toLocaleString(undefined, { maximumFractionDigits: 3 });
  return escapeHtml(`${n > 0 ? "+" : ""}${s}%`);
}

function fmtMcap(v) {
  if (v === "∞" || v === "Infinity") return "∞";
  if (v === null || v === undefined || v === "") return "—";
  const n = Number(v);
  if (Number.isNaN(n)) return escapeHtml(String(v));
  if (Math.abs(n) >= 1e12) return escapeHtml((n / 1e12).toFixed(2) + "T");
  if (Math.abs(n) >= 1e9) return escapeHtml((n / 1e9).toFixed(2) + "B");
  if (Math.abs(n) >= 1e6) return escapeHtml((n / 1e6).toFixed(2) + "M");
  return escapeHtml(n.toLocaleString(undefined, { maximumFractionDigits: 0 }));
}

function fmtTime(iso) {
  if (!iso) return "—";
  try {
    return new Date(iso).toLocaleString();
  } catch {
    return iso;
  }
}

function changeClass(v) {
  const n = Number(v);
  if (Number.isNaN(n) || n === 0) return "";
  return n > 0 ? "pos" : "neg";
}

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function escapeAttr(s) {
  return escapeHtml(s).replace(/'/g, "&#39;");
}

/** Format Binance product tags as HTML chips (already escaped). */
function formatTags(tags) {
  if (!Array.isArray(tags) || !tags.length) return "—";
  return tags
    .map((t) => `<span class="tag-chip">${escapeHtml(t)}</span>`)
    .join(" ");
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

function visibleDefs() {
  return columnOrder.map((id) => COL_BY_ID[id]).filter(Boolean);
}

function applyColumnLayout() {
  columnOrder = normalizeOrder(columnOrder);
  saveColumnOrder();
  renderColumnChips();
  renderHead();
  // Column add/remove/reorder must rebuild the full row HTML. In-place patch only
  // updates existing cells — it never deletes tds, which left orphan data under
  // missing headers.
  if (lastSpotData) renderSpotBody(lastSpotData, { forceFull: true });
}

function toggleColumn(id) {
  if (id === "symbol") return; // locked on
  if (isVisible(id)) {
    columnOrder = columnOrder.filter((x) => x !== id);
  } else {
    // append in catalog order among missing, after current list
    columnOrder = normalizeOrder([...columnOrder, id]);
  }
  applyColumnLayout();
}

function moveColumn(fromId, toId) {
  if (!fromId || !toId || fromId === toId) return;
  if (fromId === "symbol" || toId === "symbol") return;
  if (!isVisible(fromId) || !isVisible(toId)) return;
  const next = columnOrder.filter((id) => id !== fromId);
  const toIdx = next.indexOf(toId);
  if (toIdx < 0) return;
  next.splice(toIdx, 0, fromId);
  columnOrder = normalizeOrder(next);
  applyColumnLayout();
}

/** Move a visible column one step left (-1) or right (+1). Symbol stays first. */
function nudgeColumn(id, dir) {
  if (!id || id === "symbol" || !isVisible(id)) return;
  const idx = columnOrder.indexOf(id);
  if (idx < 0) return;
  const target = idx + dir;
  // Index 0 is always symbol; never move past it or out of bounds.
  if (target < 1 || target >= columnOrder.length) return;
  const next = columnOrder.slice();
  const [item] = next.splice(idx, 1);
  next.splice(target, 0, item);
  columnOrder = normalizeOrder(next);
  applyColumnLayout();
}

/** Chips: visible columns first (table order), then hidden columns (catalog order). */
function chipDisplayOrder() {
  const visible = columnOrder.slice();
  const hidden = ALL_IDS.filter((id) => !visible.includes(id));
  return [...visible, ...hidden];
}

function clearDropTargets() {
  document.querySelectorAll(".drop-target").forEach((el) => el.classList.remove("drop-target"));
}

function bindColumnDrag(el, id, { onDrop } = {}) {
  if (!el || id === "symbol") return;
  el.addEventListener("dragstart", (e) => {
    if (!isVisible(id)) {
      e.preventDefault();
      return;
    }
    dragFromId = id;
    el.dataset.dragging = "1";
    el.classList.add("dragging");
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("text/plain", id);
  });
  el.addEventListener("dragend", () => {
    el.classList.remove("dragging");
    setTimeout(() => {
      el.dataset.dragging = "0";
    }, 50);
    dragFromId = null;
    clearDropTargets();
  });
  el.addEventListener("dragover", (e) => {
    if (!dragFromId || id === "symbol" || !isVisible(id)) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    el.classList.add("drop-target");
  });
  el.addEventListener("dragleave", () => el.classList.remove("drop-target"));
  el.addEventListener("drop", (e) => {
    e.preventDefault();
    el.classList.remove("drop-target");
    const from = e.dataTransfer.getData("text/plain") || dragFromId;
    if (onDrop) onDrop(from, id);
    else moveColumn(from, id);
    dragFromId = null;
  });
}

function renderColumnChips() {
  // Visible chips follow table column order; hidden chips follow catalog order.
  els.columnChips.innerHTML = chipDisplayOrder()
    .map((colId) => {
      const col = COL_BY_ID[colId];
      if (!col) return "";
      const on = isVisible(col.id);
      const locked = col.id === "symbol";
      const visIdx = columnOrder.indexOf(col.id);
      const canLeft = on && !locked && visIdx > 1;
      const canRight = on && !locked && visIdx >= 1 && visIdx < columnOrder.length - 1;
      const classes = [
        "col-chip",
        on ? "on" : "off",
        locked ? "locked" : "",
        col.sortKey ? "sortable-col" : "display-only",
      ]
        .filter(Boolean)
        .join(" ");
      const title = locked
        ? "Symbol is always shown first"
        : on
          ? `“${col.label}”: click label to hide · drag or use ◀ ▶ to reorder`
          : `Click to show “${col.label}”.`;
      const drag = on && !locked ? 'draggable="true"' : "";
      const moveBtns =
        on && !locked
          ? `<span class="chip-move" role="group" aria-label="Reorder ${escapeAttr(col.label)}">
              <button type="button" class="chip-nudge" data-nudge="-1" data-col="${col.id}" title="Move left" ${canLeft ? "" : "disabled"} aria-label="Move ${escapeAttr(col.label)} left">◀</button>
              <button type="button" class="chip-nudge" data-nudge="1" data-col="${col.id}" title="Move right" ${canRight ? "" : "disabled"} aria-label="Move ${escapeAttr(col.label)} right">▶</button>
            </span>`
          : "";
      return `
      <div class="${classes}" data-col="${col.id}" role="listitem" title="${escapeAttr(title)}" ${drag}>
        <button type="button" class="chip-toggle" data-col="${col.id}" aria-pressed="${on}">
          <span class="chip-state">${on ? "✓" : "+"}</span>
          <span class="chip-label">${escapeHtml(col.label)}</span>
          ${col.sortKey ? "" : '<span class="chip-tag">view</span>'}
        </button>
        ${moveBtns}
      </div>`;
    })
    .join("");

  const visible = columnOrder.length;
  const total = COLUMN_DEFS.length;
  els.columnSummary.textContent = `${visible} of ${total} columns visible · order: ${columnOrder
    .map((id) => COL_BY_ID[id]?.label || id)
    .join(" → ")} · drag chips or headers, or use ◀ ▶`;

  els.columnChips.querySelectorAll(".col-chip").forEach((chip) => {
    const id = chip.getAttribute("data-col");
    const toggle = chip.querySelector(".chip-toggle");
    if (toggle) {
      toggle.addEventListener("click", (e) => {
        e.stopPropagation();
        if (chip.dataset.dragging === "1") {
          chip.dataset.dragging = "0";
          return;
        }
        toggleColumn(id);
      });
    }
    chip.querySelectorAll(".chip-nudge").forEach((btn) => {
      btn.addEventListener("click", (e) => {
        e.stopPropagation();
        e.preventDefault();
        const dir = Number(btn.getAttribute("data-nudge")) || 0;
        nudgeColumn(id, dir);
      });
    });
    if (isVisible(id) && id !== "symbol") {
      bindColumnDrag(chip, id);
    }
  });
}

function renderHead() {
  const cols = visibleDefs();
  els.spotHead.innerHTML = cols
    .map((col) => {
      const sortable = Boolean(col.sortKey);
      const active = sortable && sortState.sort === col.sortKey;
      const arrow = active ? (sortState.order === "asc" ? " ▲" : " ▼") : "";
      const locked = col.id === "symbol";
      const cls = [
        col.align === "left" ? "th-left" : "",
        sortable ? "sortable" : "not-sortable",
        active ? "sorted" : "",
        locked ? "th-locked" : "th-reorder",
      ]
        .filter(Boolean)
        .join(" ");
      const title = locked
        ? `${col.label} (always first)`
        : sortable
          ? `${col.label} — click to sort · drag to reorder`
          : `${col.label} — drag to reorder`;
      const drag = locked ? "" : 'draggable="true"';
      const grip = locked ? "" : '<span class="th-grip" aria-hidden="true" title="Drag to reorder">⋮⋮</span>';
      const sortAttr = sortable ? `data-sort="${col.sortKey}"` : "";
      return `<th class="${cls}" data-col="${col.id}" ${sortAttr} title="${escapeAttr(title)}" scope="col" role="columnheader" ${sortable || !locked ? 'tabindex="0"' : ""} ${drag}>${grip}<span class="th-label">${escapeHtml(col.label)}${arrow}</span></th>`;
    })
    .join("");

  els.spotHead.querySelectorAll("th").forEach((th) => {
    const id = th.getAttribute("data-col");
    const sortKey = th.getAttribute("data-sort");

    if (sortKey) {
      const activate = () => {
        if (th.dataset.dragging === "1") {
          th.dataset.dragging = "0";
          return;
        }
        if (sortState.sort === sortKey) {
          sortState.order = sortState.order === "desc" ? "asc" : "desc";
        } else {
          sortState.sort = sortKey;
          sortState.order =
            sortKey === "symbol" || sortKey === "baseAsset" || sortKey === "tags"
              ? "asc"
              : "desc";
        }
        saveSortState();
        loadMarkets();
      };
      th.addEventListener("click", activate);
      th.addEventListener("keydown", (e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          activate();
        }
      });
    }

    // Header drag-to-reorder (all non-symbol columns).
    if (id && id !== "symbol") {
      bindColumnDrag(th, id);
      th.addEventListener("keydown", (e) => {
        if (e.key === "ArrowLeft") {
          e.preventDefault();
          nudgeColumn(id, -1);
        } else if (e.key === "ArrowRight") {
          e.preventDefault();
          nudgeColumn(id, 1);
        }
      });
    }
  });
}

function snapshotRow(m) {
  return {
    lastPrice: m.lastPrice,
    priceChangePercent: m.priceChangePercent,
    volume: m.volume,
    quoteVolume: m.quoteVolume,
    tradeCount: m.tradeCount,
    marketCapCirculating: m.marketCapCirculating,
    marketCapTotal: m.marketCapTotal,
    marketCapMax: m.marketCapMax,
  };
}

function bindRowClicks() {
  els.spotBody.querySelectorAll("[data-pick]").forEach((btn) => {
    btn.addEventListener("click", (e) => {
      e.stopPropagation();
      selectSymbol(btn.getAttribute("data-pick"));
    });
  });
  els.spotBody.querySelectorAll("tr[data-symbol]").forEach((tr) => {
    tr.addEventListener("click", () => selectSymbol(tr.getAttribute("data-symbol")));
  });
}

/** Format a cell's inner HTML, with extra treatment for lastPrice ticks. */
function formatCellInner(col, m, move) {
  if (col.id === "lastPrice") {
    const arrow = move === "up" ? "▲" : move === "down" ? "▼" : "";
    const arrowHtml = arrow
      ? `<span class="tick-arrow" aria-hidden="true">${arrow}</span>`
      : `<span class="tick-arrow tick-arrow-empty" aria-hidden="true"></span>`;
    return `${arrowHtml}<span class="tick-value">${col.format(m)}</span>`;
  }
  return col.format(m);
}

/**
 * Apply / restart a CSS flash on an element. Always removes then re-adds on next frames
 * so consecutive polls with the same direction still animate.
 */
function playFlash(el, move) {
  if (!el || (move !== "up" && move !== "down")) return;
  el.classList.remove("flash-up", "flash-down", "flash-up-strong", "flash-down-strong");
  // Force style recalc so re-adding the class restarts the animation.
  // eslint-disable-next-line no-unused-expressions
  void el.offsetWidth;
  const strong = el.dataset.field === "lastPrice";
  const cls =
    move === "up"
      ? strong
        ? "flash-up-strong"
        : "flash-up"
      : strong
        ? "flash-down-strong"
        : "flash-down";
  // Double rAF: more reliable after innerHTML/text updates in Chromium.
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      el.classList.add(cls);
    });
  });
}

function canPatchRows(data) {
  const rows = els.spotBody.querySelectorAll("tr[data-symbol]");
  if (!rows.length || !data?.items?.length) return false;
  if (rows.length !== data.items.length) return false;
  for (let i = 0; i < data.items.length; i++) {
    if (rows[i].getAttribute("data-symbol") !== data.items[i].symbol) return false;
  }
  // Visible columns must match the DOM exactly (count + order + ids).
  // Only checking "visible fields still exist" allowed REMOVED columns' <td>s to stay.
  const cols = visibleDefs().filter((c) => c.id !== "symbol");
  const sample = rows[0];
  const fieldCells = [...sample.querySelectorAll("td[data-field]")];
  if (fieldCells.length !== cols.length) return false;
  for (let i = 0; i < cols.length; i++) {
    if (fieldCells[i].getAttribute("data-field") !== cols[i].id) return false;
  }
  return true;
}

/** Live path: update cells in place so price flashes are reliable. */
function patchSpotBody(data) {
  const prev = prevRowSnapshot;
  const nextSnap = new Map();
  const rows = els.spotBody.querySelectorAll("tr[data-symbol]");

  data.items.forEach((m, i) => {
    const tr = rows[i];
    const old = prev.get(m.symbol);
    nextSnap.set(m.symbol, snapshotRow(m));
    tr.classList.toggle("selected-row", m.symbol === selectedSymbol);

    visibleDefs().forEach((col) => {
      if (col.id === "symbol") return;
      const cell = tr.querySelector(`[data-field="${col.id}"]`);
      if (!cell) return;
      const move = old ? priceMove(old[col.id], m[col.id]) : "";
      const tone = col.id === "priceChangePercent" ? changeClass(m.priceChangePercent) : "";
      cell.className = [
        col.align === "left" ? "td-left" : "",
        tone,
        ANIMATED_FIELDS.has(col.id) ? "price-cell" : "",
        col.id === "lastPrice" ? "last-price-cell" : "",
      ]
        .filter(Boolean)
        .join(" ");
      cell.dataset.field = col.id;
      cell.innerHTML = formatCellInner(col, m, move);
      if (move) playFlash(cell, move);
    });
  });

  prevRowSnapshot = nextSnap;
}

/**
 * @param {object} data
 * @param {{ forceFull?: boolean }} [opts]
 */
function renderSpotBody(data, opts = {}) {
  lastSpotData = data;
  const cols = visibleDefs();
  const colCount = Math.max(cols.length, 1);

  if (!data || !Array.isArray(data.items)) {
    els.spotBody.innerHTML = `<tr><td colspan="${colCount}" class="muted">No data.</td></tr>`;
    els.spotMeta.textContent = "";
    prevRowSnapshot = new Map();
    return;
  }

  const liveNote =
    liveSeconds > 0 ? ` · live ${liveSeconds}s` : " · manual refresh";
  const mcapNote =
    data.sort && String(data.sort).startsWith("marketCap")
      ? " · mcap = price × daily supply snapshot"
      : " · supply/mcap from daily snapshot";
  els.spotMeta.textContent = `· ${data.total} markets · showing ${data.items.length} · sort ${data.sort} ${data.order}${liveNote}${mcapNote}`;

  if (!data.items.length) {
    els.spotBody.innerHTML = `<tr><td colspan="${colCount}" class="muted">No matches.</td></tr>`;
    prevRowSnapshot = new Map();
    return;
  }

  // Prefer in-place patch so Last price can flash without a full tbody rebuild.
  // Never patch when column layout changed (forceFull) or DOM columns diverge.
  if (!opts.forceFull && canPatchRows(data)) {
    patchSpotBody(data);
    return;
  }

  const prev = prevRowSnapshot;
  const nextSnap = new Map();

  els.spotBody.innerHTML = data.items
    .map((m) => {
      const selected = m.symbol === selectedSymbol ? "selected-row" : "";
      const old = prev.get(m.symbol);
      nextSnap.set(m.symbol, snapshotRow(m));
      const cells = cols
        .map((col) => {
          if (col.id === "symbol") {
            return `<td class="td-left"><button type="button" class="linkish" data-pick="${escapeAttr(m.symbol)}">${escapeHtml(m.symbol)}</button></td>`;
          }
          const move = old ? priceMove(old[col.id], m[col.id]) : "";
          const align = col.align === "left" ? "td-left" : "";
          const tone = col.id === "priceChangePercent" ? changeClass(m.priceChangePercent) : "";
          const priceCell = ANIMATED_FIELDS.has(col.id) ? "price-cell" : "";
          const lastCell = col.id === "lastPrice" ? "last-price-cell" : "";
          const cls = [align, tone, priceCell, lastCell].filter(Boolean).join(" ");
          return `<td class="${cls}" data-field="${col.id}" data-move="${move}">${formatCellInner(col, m, move)}</td>`;
        })
        .join("");
      return `<tr class="${selected}" data-symbol="${escapeAttr(m.symbol)}">${cells}</tr>`;
    })
    .join("");

  prevRowSnapshot = nextSnap;
  bindRowClicks();

  // Start flashes after nodes are in the DOM (do NOT strip classes afterward).
  els.spotBody.querySelectorAll("td[data-move]").forEach((td) => {
    const move = td.getAttribute("data-move");
    if (move === "up" || move === "down") playFlash(td, move);
  });
}

/**
 * @param {{ silent?: boolean }} [opts]
 */
async function loadMarkets(opts = {}) {
  const silent = Boolean(opts.silent);
  const seq = ++loadSeq;
  if (!silent) {
    els.btnRefresh.disabled = true;
    setStatus("Loading markets…");
  }
  try {
    const params = new URLSearchParams();
    const q = els.spotQ.value.trim();
    const quote = els.spotQuote.value.trim();
    const exchange = els.spotExchange?.value || "binance";
    params.set("exchange", exchange);
    if (q) params.set("q", q);
    if (quote) params.set("quote", quote);
    const tag = els.spotTag?.value?.trim() || "";
    if (tag) params.set("tag", tag);
    params.set("sort", sortState.sort);
    params.set("order", sortState.order);
    params.set("limit", els.spotLimit.value || "50");
    params.set("status", "TRADING");

    const data = await apiGet(`/api/v1/market/spot?${params.toString()}`);
    if (seq !== loadSeq) return;
    // Only rebuild headers when not a quiet poll (avoids focus steal / flicker)
    if (!silent) renderHead();
    else if (!els.spotHead.children.length) renderHead();

    const fingerprint = spotFingerprint(data);
    const pricesMoved = fingerprint !== lastSpotFingerprint && lastSpotFingerprint !== "";
    lastSpotFingerprint = fingerprint;

    renderSpotBody(data);
    const t = new Date().toLocaleTimeString();
    if (silent) {
      setStatus(
        pricesMoved
          ? `Live · prices moved · ${t} · next ~${liveSeconds}s`
          : `Live · polled ${t} · waiting for new ticks · next ~${liveSeconds}s`,
        "ok"
      );
    } else {
      setStatus(`Updated ${t}`, "ok");
    }
    // Keep detail ticker relatively fresh while live
    if (silent && selectedSymbol && !els.detailSection.hidden) {
      refreshDetailTickerQuiet();
    }
  } catch (err) {
    if (seq !== loadSeq) return;
    console.error(err);
    if (!silent) {
      setStatus(String(err.message || err), "error");
      const cols = Math.max(visibleDefs().length, 1);
      els.spotBody.innerHTML = `<tr><td colspan="${cols}" class="muted">Failed to load. Is the API running at ${escapeHtml(baseUrl())}?</td></tr>`;
    } else {
      setStatus(`Live update failed: ${String(err.message || err)}`, "error");
    }
  } finally {
    if (seq === loadSeq && !silent) els.btnRefresh.disabled = false;
  }
}

async function refreshDetailTickerQuiet() {
  if (!selectedSymbol) return;
  try {
    const exchange = els.spotExchange?.value || "binance";
    const ticker = await apiGet(
      `/api/v1/market/ticker/24h?exchange=${encodeURIComponent(exchange)}&symbol=${encodeURIComponent(selectedSymbol)}`
    );
    renderTicker(ticker);
    if (els.ticker) els.ticker.dataset.stale = "0";
  } catch (err) {
    if (els.ticker) {
      els.ticker.dataset.stale = "1";
      const note = document.createElement("p");
      note.className = "muted";
      note.dataset.staleNote = "1";
      note.textContent = `Ticker may be stale · ${String(err.message || err)}`;
      // Replace prior note only
      els.ticker.querySelectorAll("[data-stale-note]").forEach((n) => n.remove());
      els.ticker.appendChild(note);
    }
  }
}

function scheduleSearch() {
  clearTimeout(searchDebounce);
  searchDebounce = setTimeout(() => loadMarkets(), 280);
}

async function selectSymbol(sym) {
  if (!sym) return;
  selectedSymbol = sym;
  els.detailSymbol.textContent = sym;
  els.detailSection.hidden = false;
  els.selectedHint.textContent = `Selected ${sym}`;
  if (lastSpotData) renderSpotBody(lastSpotData);
  await loadDetail();
}

async function loadDetail() {
  if (!selectedSymbol) return;
  const sym = selectedSymbol;
  setStatus(`Loading detail for ${sym}…`);
  try {
    const exchange = els.spotExchange?.value || "binance";
    const [ticker, supply, candles] = await Promise.all([
      apiGet(`/api/v1/market/ticker/24h?exchange=${encodeURIComponent(exchange)}&symbol=${encodeURIComponent(sym)}`),
      apiGet(`/api/v1/market/supply?symbol=${encodeURIComponent(sym)}`).catch((e) => ({
        _error: String(e.message || e),
      })),
      apiGet(
        `/api/v1/market/candles?exchange=${encodeURIComponent(exchange)}&symbol=${encodeURIComponent(sym)}&interval=${encodeURIComponent(els.interval.value)}&limit=${encodeURIComponent(els.limit.value || "24")}`
      ),
    ]);
    renderTicker(ticker);
    if (supply?._error) {
      els.supply.innerHTML = `<span class="muted">${escapeHtml(supply._error)}</span>`;
    } else {
      renderSupply(supply);
    }
    renderCandles(candles);
    setStatus(`Detail ready · ${sym}`, "ok");
  } catch (err) {
    console.error(err);
    setStatus(String(err.message || err), "error");
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
      <dt>High / Low</dt><dd>${fmtNum(data.highPrice, 8)} / ${fmtNum(data.lowPrice, 8)}</dd>
      <dt>Volume</dt><dd>${fmtNum(data.volume, 4)}</dd>
      <dt>Quote vol</dt><dd>${fmtNum(data.quoteVolume, 2)}</dd>
      <dt>Trades</dt><dd>${fmtNum(data.tradeCount, 0)}</dd>
    </dl>
  `;
}

function renderSupply(data) {
  if (!data) {
    els.supply.innerHTML = `<span class="muted">—</span>`;
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
      <dt>As of</dt><dd>${escapeHtml(asOf)} <span class="muted">(daily snapshot, not live)</span></dd>
      <dt>Source</dt><dd>${escapeHtml(data.source || "—")}</dd>
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
        <td>${fmtNum(c.tradeCount, 0)}</td>
      </tr>`
    )
    .join("");
}

// —— Wire UI ——
els.btnRefresh.addEventListener("click", () => loadMarkets({ silent: false }));
els.spotQ.addEventListener("input", scheduleSearch);
els.spotQ.addEventListener("keydown", (e) => {
  if (e.key === "Enter") {
    clearTimeout(searchDebounce);
    loadMarkets();
  }
});
els.spotQuote.addEventListener("change", () => loadMarkets());
if (els.spotTag) els.spotTag.addEventListener("change", () => loadMarkets());
if (els.spotExchange) {
  els.spotExchange.addEventListener("change", async () => {
    const ex = els.spotExchange.value;
    if (ex === "coinbase" && (els.spotQuote.value === "USDT" || !els.spotQuote.value)) {
      els.spotQuote.value = "USD";
    } else if (ex !== "coinbase" && els.spotQuote.value === "USD") {
      els.spotQuote.value = "USDT";
    }
    try {
      const data = await apiGet(`/api/v1/market/intervals?exchange=${encodeURIComponent(ex)}`);
      if (Array.isArray(data.intervals) && data.intervals.length && els.interval) {
        const current = els.interval.value;
        els.interval.innerHTML = data.intervals
          .map((iv) => `<option value="${escapeAttr(iv)}">${escapeHtml(iv)}</option>`)
          .join("");
        els.interval.value = data.intervals.includes(current) ? current : data.intervals[0];
      }
    } catch { /* ignore */ }
    if (els.spotTag) {
      els.spotTag.value = "";
      els.spotTag.disabled = ex !== "binance";
    }
    loadMarkets();
  });
}
els.spotLimit.addEventListener("change", () => loadMarkets());
els.apiBase.addEventListener("change", () => loadMarkets());

if (els.liveInterval) {
  els.liveInterval.value = String(liveSeconds);
  els.liveInterval.addEventListener("change", () => {
    liveSeconds = Number(els.liveInterval.value) || 0;
    saveLiveSeconds();
    startPolling();
    loadMarkets({ silent: true });
  });
}

document.addEventListener("visibilitychange", () => {
  updateLiveBadge();
  if (!document.hidden && liveSeconds > 0) {
    loadMarkets({ silent: true });
  }
});

els.btnColumnsAll.addEventListener("click", () => {
  columnOrder = normalizeOrder(ALL_IDS);
  applyColumnLayout();
});
els.btnColumnsDefaults.addEventListener("click", () => {
  columnOrder = defaultOrder();
  applyColumnLayout();
});
els.btnColumnsMinimal.addEventListener("click", () => {
  columnOrder = minimalOrder();
  applyColumnLayout();
});

els.btnDetailClose.addEventListener("click", () => {
  selectedSymbol = null;
  els.detailSection.hidden = true;
  els.selectedHint.textContent = "Click a symbol for detail (ticker / supply / candles).";
  if (lastSpotData) renderSpotBody(lastSpotData);
});
els.btnDetailRefresh.addEventListener("click", () => loadDetail());
els.interval.addEventListener("change", () => {
  if (selectedSymbol) loadDetail();
});
els.limit.addEventListener("change", () => {
  if (selectedSymbol) loadDetail();
});

(async () => {
  try {
    const data = await apiGet("/api/v1/market/intervals");
    if (Array.isArray(data.intervals) && data.intervals.length) {
      const current = els.interval.value;
      els.interval.innerHTML = data.intervals
        .map((iv) => `<option value="${escapeAttr(iv)}">${escapeHtml(iv)}</option>`)
        .join("");
      els.interval.value = data.intervals.includes(current) ? current : "1h";
    }
  } catch {
    /* backend may not be up yet */
  }
  try {
    const data = await apiGet("/api/v1/market/exchanges");
    if (els.spotExchange && Array.isArray(data.exchanges) && data.exchanges.length) {
      const current = els.spotExchange.value || data.default || "binance";
      els.spotExchange.innerHTML = data.exchanges
        .map((ex) => `<option value="${escapeAttr(ex)}">${escapeHtml(ex)}</option>`)
        .join("");
      els.spotExchange.value = data.exchanges.includes(current) ? current : (data.default || data.exchanges[0]);
    }
  } catch { /* optional */ }
  try {
    const data = await apiGet("/api/v1/market/tags");
    if (els.spotTag && Array.isArray(data.tags) && data.tags.length) {
      const current = els.spotTag.value;
      const opts = ['<option value="">All tags</option>'].concat(
        data.tags.map((t) => `<option value="${escapeAttr(t)}">${escapeHtml(t)}</option>`)
      );
      els.spotTag.innerHTML = opts.join("");
      if (current && data.tags.includes(current)) els.spotTag.value = current;
    }
  } catch {
    /* tags optional until backend is up */
  }
})();

renderColumnChips();
renderHead();
updateLiveBadge();
startPolling();
loadMarkets();
