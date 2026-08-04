/**
 * node --test simple-frontend/markdown.test.js
 */
const { describe, it } = require("node:test");
const assert = require("node:assert/strict");

// Load UMD-style script
require("./markdown.js");
const { render, escapeHtml } = globalThis.SwyngoraMarkdown;

describe("SwyngoraMarkdown", () => {
  it("escapes raw HTML", () => {
    const html = render('<script>alert(1)</script>');
    assert.ok(!html.includes("<script>"));
    assert.ok(html.includes("&lt;script&gt;"));
  });

  it("renders bold and inline code", () => {
    const html = render("Hello **world** and `code`");
    assert.ok(html.includes("<strong>world</strong>"));
    assert.ok(html.includes('<code class="md-code">code</code>'));
  });

  it("renders fenced code blocks", () => {
    const html = render("```js\nconst x = 1;\n```");
    assert.ok(html.includes("<pre class=\"md-pre\">"));
    assert.ok(html.includes("const x = 1;"));
  });

  it("renders headings and lists", () => {
    const html = render("# Title\n\n- a\n- b\n");
    assert.ok(html.includes("<h1"));
    assert.ok(html.includes("<ul"));
    assert.ok(html.includes("<li>"));
  });

  it("renders tables", () => {
    const md = "| A | B |\n| --- | --- |\n| 1 | 2 |\n";
    const html = render(md);
    assert.ok(html.includes("<table"));
    assert.ok(html.includes("<th"));
    assert.ok(html.includes("1"));
  });

  it("blocks javascript: links", () => {
    const html = render("[x](javascript:alert(1))");
    assert.ok(!html.includes("javascript:"));
  });

  it("escapeHtml is available", () => {
    assert.equal(escapeHtml("<a>"), "&lt;a&gt;");
  });

  it("strips orphan bold markers from truncated output", () => {
    const html = render("RSI(14): **6…");
    assert.ok(!html.includes("**"));
    assert.ok(html.includes("6…") || html.includes("6…") || html.includes("6"));
  });

  it("renders multiple bold spans on one line", () => {
    const html = render(
      "**BTCUSDT (Binance, 1h)** - Latest close: **64089.99** - RSI(14): **60.36**"
    );
    assert.equal((html.match(/<strong>/g) || []).length, 3);
    assert.ok(!html.includes("**"));
  });

  it("renderInline applies bold", () => {
    const { renderInline } = globalThis.SwyngoraMarkdown;
    const html = renderInline("close **64089.99** now");
    assert.ok(html.includes("<strong>64089.99</strong>"));
    assert.ok(!html.includes("**"));
  });
});
