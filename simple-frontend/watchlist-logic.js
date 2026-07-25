/**
 * Pure watchlist helpers (shared by UI + unit tests).
 * Browser: attaches to globalThis.WatchlistLogic
 * Node: module.exports
 */
(function (root, factory) {
  const api = factory();
  if (typeof module !== "undefined" && module.exports) {
    module.exports = api;
  }
  root.WatchlistLogic = api;
})(typeof globalThis !== "undefined" ? globalThis : this, function () {
  "use strict";

  function toNum(v) {
    if (v === null || v === undefined || v === "" || v === "∞" || v === "Infinity") return null;
    const n = Number(v);
    return Number.isFinite(n) ? n : null;
  }

  /**
   * Which watchlist entries should be loaded.
   * @param {{exchange:string,symbol:string}[]} watchlist
   * @param {{ currentExchangeOnly?: boolean, currentExchange?: string, search?: string }} opts
   */
  function filterWatchlistTargets(watchlist, opts = {}) {
    let targets = Array.isArray(watchlist) ? watchlist.slice() : [];
    if (opts.currentExchangeOnly) {
      const ex = String(opts.currentExchange || "binance").toLowerCase();
      targets = targets.filter(
        (w) => String(w.exchange || "binance").toLowerCase() === ex
      );
    }
    const q = String(opts.search || "")
      .trim()
      .toUpperCase();
    if (q) {
      targets = targets.filter((w) =>
        String(w.symbol || "")
          .toUpperCase()
          .includes(q)
      );
    }
    // Dedupe by exchange|symbol
    const seen = new Set();
    const out = [];
    for (const w of targets) {
      const key =
        String(w.exchange || "binance").toLowerCase() +
        "|" +
        String(w.symbol || "").toUpperCase();
      if (seen.has(key)) continue;
      seen.add(key);
      out.push({
        exchange: String(w.exchange || "binance").toLowerCase(),
        symbol: String(w.symbol || "").toUpperCase(),
      });
    }
    return out;
  }

  /** Exact symbol match from a spot list response (avoids q=BTC matching BTCUSDC only issues). */
  function pickExactSymbol(items, symbol) {
    const sym = String(symbol || "").toUpperCase();
    if (!Array.isArray(items) || !sym) return null;
    return (
      items.find((m) => String(m.symbol || "").toUpperCase() === sym) || null
    );
  }

  /**
   * Build display rows for every target. Missing markets become placeholders
   * so membership length stays equal to targets.length (stable count).
   */
  function buildWatchlistRows(targets, resolvedByKey) {
    const rows = [];
    for (const t of targets) {
      const key = t.exchange + "|" + t.symbol;
      const hit = resolvedByKey && resolvedByKey[key];
      if (hit) {
        rows.push({ ...hit, _watchExchange: t.exchange, _missing: false });
      } else {
        rows.push({
          symbol: t.symbol,
          lastPrice: "",
          priceChangePercent: "",
          volume: "",
          quoteVolume: "",
          tradeCount: 0,
          tags: [],
          _missing: true,
          _watchExchange: t.exchange,
        });
      }
    }
    return rows;
  }

  /**
   * Wrong approach (regression fixture): filter a paged market list by watchlist.
   * This drops coins that are not on the current page — the bug we fixed.
   */
  function filterPageByWatchlist(pageItems, watchlist, exchange) {
    const ex = String(exchange || "binance").toLowerCase();
    const keys = new Set(
      (watchlist || [])
        .filter((w) => String(w.exchange || "binance").toLowerCase() === ex)
        .map((w) => String(w.symbol || "").toUpperCase())
    );
    return (pageItems || []).filter((m) =>
      keys.has(String(m.symbol || "").toUpperCase())
    );
  }

  function sortSpotItemsClient(items, sort, order) {
    const desc = order === "desc";
    const key = sort || "quoteVolume";
    const strKeys = new Set(["symbol", "tags"]);
    return [...(items || [])].sort((a, b) => {
      let cmp = 0;
      if (key === "tags") {
        const as = (Array.isArray(a.tags) ? a.tags.join(",") : "").toLowerCase();
        const bs = (Array.isArray(b.tags) ? b.tags.join(",") : "").toLowerCase();
        cmp = as.localeCompare(bs);
      } else if (strKeys.has(key)) {
        cmp = String(a[key] || "").localeCompare(String(b[key] || ""), undefined, {
          sensitivity: "base",
        });
      } else if (key === "tradeCount") {
        const av = Number(a.tradeCount) || 0;
        const bv = Number(b.tradeCount) || 0;
        cmp = av < bv ? -1 : av > bv ? 1 : 0;
      } else {
        const av = toNum(a[key]);
        const bv = toNum(b[key]);
        // Nulls always last (independent of asc/desc).
        if (av === null && bv === null) {
          return String(a.symbol || "").localeCompare(String(b.symbol || ""));
        }
        if (av === null) return 1;
        if (bv === null) return -1;
        cmp = av < bv ? -1 : av > bv ? 1 : 0;
      }
      if (cmp === 0) {
        return String(a.symbol || "").localeCompare(String(b.symbol || ""));
      }
      return desc ? -cmp : cmp;
    });
  }

  /**
   * Full pure pipeline for watchlist-only view.
   * @returns {{ items: object[], total: number }}
   */
  function assembleWatchlistView(watchlist, resolvedByKey, opts = {}) {
    const targets = filterWatchlistTargets(watchlist, opts);
    let rows = buildWatchlistRows(targets, resolvedByKey);
    if (opts.tag) {
      const tag = String(opts.tag).toLowerCase();
      rows = rows.filter(
        (m) =>
          Array.isArray(m.tags) &&
          m.tags.some((t) => String(t).toLowerCase() === tag)
      );
    }
    const sorted = sortSpotItemsClient(rows, opts.sort || "quoteVolume", opts.order || "desc");
    return { items: sorted, total: sorted.length, targets };
  }

  return {
    toNum,
    filterWatchlistTargets,
    pickExactSymbol,
    buildWatchlistRows,
    filterPageByWatchlist,
    sortSpotItemsClient,
    assembleWatchlistView,
  };
});
