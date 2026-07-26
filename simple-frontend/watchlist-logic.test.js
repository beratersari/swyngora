/**
 * Unit tests for watchlist pure logic.
 * Run: node --test simple-frontend/watchlist-logic.test.js
 */
const { test, describe } = require("node:test");
const assert = require("node:assert/strict");
const {
  filterWatchlistTargets,
  pickExactSymbol,
  buildWatchlistRows,
  filterPageByWatchlist,
  sortSpotItemsClient,
  assembleWatchlistView,
  mergeWatchlists,
  fmtNum,
} = require("./watchlist-logic.js");

const sixWatch = [
  { exchange: "binance", symbol: "BTCUSDT" },
  { exchange: "binance", symbol: "ETHUSDT" },
  { exchange: "binance", symbol: "SOLUSDT" },
  { exchange: "binance", symbol: "XRPUSDT" },
  { exchange: "binance", symbol: "DOGEUSDT" },
  { exchange: "binance", symbol: "ADAUSDT" },
];

const topByVolumePage = [
  { symbol: "BTCUSDT", lastPrice: "100", quoteVolume: "9000" },
  { symbol: "ETHUSDT", lastPrice: "50", quoteVolume: "8000" },
  // only 1 other watched coin on this "page"
  { symbol: "SOLUSDT", lastPrice: "20", quoteVolume: "100" },
  { symbol: "RANDOM1", lastPrice: "1", quoteVolume: "7000" },
  { symbol: "RANDOM2", lastPrice: "2", quoteVolume: "6000" },
];

describe("mergeWatchlists", () => {
  test("unions local and server without wiping offline adds", () => {
    const local = [
      { exchange: "binance", symbol: "BTCUSDT" },
      { exchange: "coinbase", symbol: "ETH-USD" },
    ];
    const server = [{ exchange: "binance", symbol: "BTCUSDT" }];
    const m = mergeWatchlists(local, server);
    assert.equal(m.length, 2);
    assert.ok(m.some((x) => x.exchange === "coinbase" && x.symbol === "ETH-USD"));
  });

  test("empty server does not clear local", () => {
    const local = [{ exchange: "binance", symbol: "SOLUSDT" }];
    const m = mergeWatchlists(local, []);
    assert.equal(m.length, 1);
    assert.equal(m[0].symbol, "SOLUSDT");
  });
});

describe("fmtNum", () => {
  test("does not collapse tiny non-zero prices to 0", () => {
    const s = fmtNum("1e-9", 8);
    assert.notEqual(s, "0");
    assert.ok(s.includes("e") || s.includes("E") || Number(s) > 0);
  });

  test("zero stays zero", () => {
    assert.equal(fmtNum(0, 8), "0");
  });
});

describe("filterWatchlistTargets", () => {
  test("keeps all 6 entries", () => {
    const t = filterWatchlistTargets(sixWatch, {});
    assert.equal(t.length, 6);
  });

  test("filters by exchange", () => {
    const mixed = [
      ...sixWatch,
      { exchange: "bybit", symbol: "BTCUSDT" },
    ];
    const t = filterWatchlistTargets(mixed, {
      currentExchangeOnly: true,
      currentExchange: "binance",
    });
    assert.equal(t.length, 6);
    assert.ok(t.every((x) => x.exchange === "binance"));
  });

  test("search filters symbols", () => {
    const t = filterWatchlistTargets(sixWatch, { search: "btc" });
    assert.equal(t.length, 1);
    assert.equal(t[0].symbol, "BTCUSDT");
  });

  test("dedupes exchange|symbol", () => {
    const t = filterWatchlistTargets(
      [
        { exchange: "binance", symbol: "btcusdt" },
        { exchange: "binance", symbol: "BTCUSDT" },
      ],
      {}
    );
    assert.equal(t.length, 1);
  });
});

describe("regression: page-filter is wrong", () => {
  test("filtering top-N page drops most watched coins (old bug)", () => {
    const filtered = filterPageByWatchlist(topByVolumePage, sixWatch, "binance");
    // Only 3 of 6 appear on the volume page — this is the bug.
    assert.equal(filtered.length, 3);
    assert.notEqual(filtered.length, sixWatch.length);
  });

  test("assembleWatchlistView keeps all 6 even if only 3 resolved", () => {
    const resolved = {
      "binance|BTCUSDT": { symbol: "BTCUSDT", lastPrice: "100", quoteVolume: "9" },
      "binance|ETHUSDT": { symbol: "ETHUSDT", lastPrice: "50", quoteVolume: "8" },
      "binance|SOLUSDT": { symbol: "SOLUSDT", lastPrice: "20", quoteVolume: "1" },
    };
    const view = assembleWatchlistView(sixWatch, resolved, {
      sort: "lastPrice",
      order: "desc",
    });
    assert.equal(view.total, 6, "must show all watchlist members");
    assert.equal(view.items.length, 6);
    assert.equal(view.items.filter((i) => i._missing).length, 3);
  });
});

describe("sort does not change membership", () => {
  test("sort by lastPrice keeps length 6", () => {
    const resolved = Object.fromEntries(
      sixWatch.map((w, i) => [
        `binance|${w.symbol}`,
        {
          symbol: w.symbol,
          lastPrice: String(10 + i * 3),
          quoteVolume: String(100 - i),
        },
      ])
    );
    const byPrice = assembleWatchlistView(sixWatch, resolved, {
      sort: "lastPrice",
      order: "desc",
    });
    const byVol = assembleWatchlistView(sixWatch, resolved, {
      sort: "quoteVolume",
      order: "desc",
    });
    assert.equal(byPrice.total, 6);
    assert.equal(byVol.total, 6);
    // order differs, membership same
    const setA = new Set(byPrice.items.map((i) => i.symbol)).size;
    const setB = new Set(byVol.items.map((i) => i.symbol)).size;
    assert.equal(setA, 6);
    assert.equal(setB, 6);
    assert.notDeepEqual(
      byPrice.items.map((i) => i.symbol),
      byVol.items.map((i) => i.symbol)
    );
  });

  test("sortSpotItemsClient orders lastPrice desc", () => {
    const items = [
      { symbol: "A", lastPrice: "10" },
      { symbol: "B", lastPrice: "30" },
      { symbol: "C", lastPrice: "20" },
    ];
    const sorted = sortSpotItemsClient(items, "lastPrice", "desc");
    assert.deepEqual(
      sorted.map((i) => i.symbol),
      ["B", "C", "A"]
    );
  });

  test("null numeric values sort last", () => {
    const items = [
      { symbol: "A", lastPrice: "" },
      { symbol: "B", lastPrice: "5" },
      { symbol: "C", lastPrice: null },
    ];
    const sorted = sortSpotItemsClient(items, "lastPrice", "desc");
    assert.equal(sorted[0].symbol, "B");
    assert.ok(["A", "C"].includes(sorted[1].symbol));
    assert.ok(["A", "C"].includes(sorted[2].symbol));
  });
});

describe("pickExactSymbol", () => {
  test("matches exact only", () => {
    const items = [
      { symbol: "BTCUSDT" },
      { symbol: "BTCUSDC" },
      { symbol: "WBTCUSDT" },
    ];
    assert.equal(pickExactSymbol(items, "BTCUSDT").symbol, "BTCUSDT");
    assert.equal(pickExactSymbol(items, "ETHUSDT"), null);
  });
});

describe("buildWatchlistRows", () => {
  test("placeholder for missing keeps stable length", () => {
    const targets = filterWatchlistTargets(sixWatch, {});
    const rows = buildWatchlistRows(targets, {
      "binance|BTCUSDT": { symbol: "BTCUSDT", lastPrice: "1" },
    });
    assert.equal(rows.length, 6);
    assert.equal(rows.filter((r) => r._missing).length, 5);
  });
});
