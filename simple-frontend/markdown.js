/**
 * Lightweight Markdown → safe HTML for AI chat bubbles.
 * Supports: headings, paragraphs, bold/italic, inline code, fenced code,
 * lists, blockquotes, links, hr, simple tables. No HTML passthrough.
 */
(function (global) {
  "use strict";

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

  /** Normalize common “smart” punctuation that breaks Markdown parsers. */
  function normalizeMarkdownSource(src) {
    return String(src || "")
      .replace(/\r\n/g, "\n")
      .replace(/\u2217|\u204e|\ufe61/g, "*") // fancy asterisks → *
      .replace(/\u201c|\u201d/g, '"')
      .replace(/\u2018|\u2019/g, "'");
  }

  /** Safe http(s) or relative # anchors only. */
  function safeHref(href) {
    const h = String(href || "").trim();
    if (!h) return "";
    if (h.startsWith("#")) return escapeAttr(h);
    try {
      const u = new URL(h, "https://example.invalid");
      if (u.protocol === "http:" || u.protocol === "https:") return escapeAttr(h);
    } catch {
      /* ignore */
    }
    return "";
  }

  /**
   * Apply **bold** / *italic* with pairing so orphan ** do not remain visible.
   * Incomplete markers at end of string are dropped.
   */
  function applyEmphasis(escaped) {
    let s = escaped;
    // Bold: **…** non-greedy, no newlines inside
    s = s.replace(/\*\*([^*\n]+?)\*\*/g, "<strong>$1</strong>");
    s = s.replace(/__([^_\n]+?)__/g, "<strong>$1</strong>");
    // Italic: single * not part of **
    s = s.replace(/(^|[^\*])\*([^*\n]+?)\*(?!\*)/g, "$1<em>$2</em>");
    s = s.replace(/(^|[^_])_([^_\n]+?)_(?!_)/g, "$1<em>$2</em>");
    // Drop remaining orphan emphasis markers (truncated LLM output)
    s = s.replace(/\*\*/g, "");
    s = s.replace(/(^|[^\w])\*(?!\*)/g, "$1");
    return s;
  }

  function inlineFormat(text) {
    let s = escapeHtml(text);
    // Inline code first (protect contents)
    const codes = [];
    s = s.replace(/`([^`\n]+)`/g, (_, code) => {
      const i = codes.length;
      codes.push(`<code class="md-code">${code}</code>`);
      return `\u0000C${i}\u0000`;
    });
    // Links [text](url)
    s = s.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, (_, label, href) => {
      const safe = safeHref(href);
      if (!safe) return label;
      return `<a href="${safe}" target="_blank" rel="noopener noreferrer">${label}</a>`;
    });
    s = applyEmphasis(s);
    // Restore code
    s = s.replace(/\u0000C(\d+)\u0000/g, (_, i) => codes[Number(i)] || "");
    return s;
  }

  function isTableSep(line) {
    return /^\s*\|?[\s:|-]+\|[\s:|-]+\|?\s*$/.test(line) && /\|/.test(line) && /-+/.test(line);
  }

  function splitTableRow(line) {
    let s = line.trim();
    if (s.startsWith("|")) s = s.slice(1);
    if (s.endsWith("|")) s = s.slice(0, -1);
    return s.split("|").map((c) => c.trim());
  }

  function renderTable(headerLine, sepLine, bodyLines) {
    const headers = splitTableRow(headerLine);
    const aligns = splitTableRow(sepLine).map((cell) => {
      const t = cell.trim();
      const left = t.startsWith(":");
      const right = t.endsWith(":");
      if (left && right) return "center";
      if (right) return "right";
      return "left";
    });
    let html = '<div class="md-table-wrap"><table class="md-table"><thead><tr>';
    headers.forEach((h, i) => {
      const a = aligns[i] || "left";
      html += `<th style="text-align:${a}">${inlineFormat(h)}</th>`;
    });
    html += "</tr></thead><tbody>";
    bodyLines.forEach((row) => {
      const cells = splitTableRow(row);
      html += "<tr>";
      headers.forEach((_, i) => {
        const a = aligns[i] || "left";
        html += `<td style="text-align:${a}">${inlineFormat(cells[i] || "")}</td>`;
      });
      html += "</tr>";
    });
    html += "</tbody></table></div>";
    return html;
  }

  function renderMarkdown(src) {
    const text = normalizeMarkdownSource(src);
    if (!text.trim()) return "";

    // Extract fenced code blocks
    const fences = [];
    const withFences = text.replace(/```([a-zA-Z0-9_+-]*)\n?([\s\S]*?)```/g, (_, lang, code) => {
      const i = fences.length;
      const cls = lang ? ` class="language-${escapeAttr(lang)}"` : "";
      fences.push(
        `<pre class="md-pre"><code${cls}>${escapeHtml(String(code).replace(/\n$/, ""))}</code></pre>`
      );
      return `\n\n%%FENCE${i}%%\n\n`;
    });

    const lines = withFences.split("\n");
    const out = [];
    let i = 0;
    let para = [];

    function flushPara() {
      if (!para.length) return;
      // Preserve soft line breaks inside a paragraph block when original had newlines
      const joined = para.join("\n").trim();
      para = [];
      if (!joined) return;
      // Single line → <p>; multi-line plain → <p> with <br>
      const parts = joined.split("\n").map((l) => inlineFormat(l));
      if (parts.length === 1) {
        out.push(`<p>${parts[0]}</p>`);
      } else {
        out.push(`<p>${parts.join("<br>")}</p>`);
      }
    }

    while (i < lines.length) {
      const line = lines[i];
      const trimmed = line.trim();

      const fenceMatch = trimmed.match(/^%%FENCE(\d+)%%$/);
      if (fenceMatch) {
        flushPara();
        out.push(fences[Number(fenceMatch[1])] || "");
        i += 1;
        continue;
      }

      if (!trimmed) {
        flushPara();
        i += 1;
        continue;
      }

      if (/^(-{3,}|\*{3,}|_{3,})$/.test(trimmed)) {
        flushPara();
        out.push('<hr class="md-hr" />');
        i += 1;
        continue;
      }

      const h = trimmed.match(/^(#{1,6})\s+(.+)$/);
      if (h) {
        flushPara();
        const level = h[1].length;
        out.push(`<h${level} class="md-h">${inlineFormat(h[2])}</h${level}>`);
        i += 1;
        continue;
      }

      if (trimmed.startsWith(">")) {
        flushPara();
        const parts = [];
        while (i < lines.length && lines[i].trim().startsWith(">")) {
          parts.push(lines[i].trim().replace(/^>\s?/, ""));
          i += 1;
        }
        out.push(`<blockquote class="md-quote">${inlineFormat(parts.join(" "))}</blockquote>`);
        continue;
      }

      if (trimmed.includes("|") && i + 1 < lines.length && isTableSep(lines[i + 1])) {
        flushPara();
        const header = lines[i];
        const sep = lines[i + 1];
        i += 2;
        const body = [];
        while (i < lines.length && lines[i].includes("|") && lines[i].trim()) {
          body.push(lines[i]);
          i += 1;
        }
        out.push(renderTable(header, sep, body));
        continue;
      }

      if (/^[-*+]\s+/.test(trimmed)) {
        flushPara();
        const items = [];
        while (i < lines.length && /^[-*+]\s+/.test(lines[i].trim())) {
          items.push(lines[i].trim().replace(/^[-*+]\s+/, ""));
          i += 1;
        }
        out.push(
          `<ul class="md-list">${items.map((it) => `<li>${inlineFormat(it)}</li>`).join("")}</ul>`
        );
        continue;
      }

      if (/^\d+\.\s+/.test(trimmed)) {
        flushPara();
        const items = [];
        while (i < lines.length && /^\d+\.\s+/.test(lines[i].trim())) {
          items.push(lines[i].trim().replace(/^\d+\.\s+/, ""));
          i += 1;
        }
        out.push(
          `<ol class="md-list">${items.map((it) => `<li>${inlineFormat(it)}</li>`).join("")}</ol>`
        );
        continue;
      }

      // Bullet-like lines from agents using "• " or "– "
      if (/^[•–—]\s+/.test(trimmed)) {
        flushPara();
        const items = [];
        while (i < lines.length && /^[•–—*-]\s+/.test(lines[i].trim())) {
          items.push(lines[i].trim().replace(/^[•–—*-]\s+/, ""));
          i += 1;
        }
        out.push(
          `<ul class="md-list">${items.map((it) => `<li>${inlineFormat(it)}</li>`).join("")}</ul>`
        );
        continue;
      }

      para.push(trimmed);
      i += 1;
    }
    flushPara();
    return out.join("\n");
  }

  /** Inline-only markdown (one line / chip text). */
  function renderInline(src) {
    return inlineFormat(normalizeMarkdownSource(src).replace(/\n/g, " ").trim());
  }

  global.SwyngoraMarkdown = {
    render: renderMarkdown,
    renderInline,
    escapeHtml,
    normalizeMarkdownSource,
  };
})(typeof window !== "undefined" ? window : globalThis);
