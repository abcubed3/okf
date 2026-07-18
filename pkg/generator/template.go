// Package generator compiles an OKF bundle into an interactive static web application.
package generator

// HTMLTemplate is the single-page HTML, CSS, and JS application template for rendering
// the interactive documentation catalog. It follows the viz.html design pattern from
// https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf/bundles —
// using Cytoscape.js for graph rendering as the primary navigation interface, a sidebar
// for list-based navigation, and a right-hand detail panel for concept content.
const HTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} — OKF Documentation</title>

    {{.JSONLD}}

    <!-- Google Fonts -->
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=Fira+Code:wght@400;500&display=swap" rel="stylesheet">

    <!-- Cytoscape.js — graph rendering -->
    <script src="https://cdn.jsdelivr.net/npm/cytoscape@3.28.1/dist/cytoscape.min.js"></script>
    <!-- marked.js — markdown rendering (SRI-pinned) -->
    <script src="https://cdn.jsdelivr.net/npm/marked@4.3.0/marked.min.js"
            integrity="sha256-xoB1Zy2Xbkd3OQVguqESGUhVvUQEsTZH2khVquH5Ngw="
            crossorigin="anonymous"></script>

    <style>
        /* ── Reset & Box Model ──────────────────────────────────── */
        * { box-sizing: border-box; margin: 0; padding: 0; }

        /* ── Design Tokens ──────────────────────────────────────── */
        :root {
            --bg:        #f8fafc;
            --surface:   #ffffff;
            --border:    #e2e8f0;
            --text:      #0f172a;
            --muted:     #64748b;
            --accent:    #2563eb;
            --accent-bg: #eff6ff;
            --code-bg:   #f1f5f9;
            --shadow:    0 1px 3px rgba(0,0,0,0.08);
            --radius:    6px;
            --sidebar-w: 260px;

            /* Type colors — matches viz.html palette */
            --c-dataset:   #8b5cf6;
            --c-table:     #3b82f6;
            --c-metric:    #10b981;
            --c-reference: #10b981;
            --c-playbook:  #f59e0b;
            --c-default:   #94a3b8;
        }

        html.dark {
            --bg:        #0f172a;
            --surface:   #1e293b;
            --border:    #334155;
            --text:      #f1f5f9;
            --muted:     #94a3b8;
            --accent:    #60a5fa;
            --accent-bg: #1e3a5f;
            --code-bg:   #0f172a;
            --shadow:    0 1px 3px rgba(0,0,0,0.4);
        }

        /* ── Base ───────────────────────────────────────────────── */
        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif;
            font-size: 14px;
            color: var(--text);
            background: var(--bg);
            display: flex;
            flex-direction: column;
            height: 100vh;
            overflow: hidden;
        }

        /* ── Header ─────────────────────────────────────────────── */
        header {
            display: flex;
            align-items: center;
            gap: 10px;
            padding: 8px 16px;
            background: var(--surface);
            border-bottom: 1px solid var(--border);
            flex-shrink: 0;
            box-shadow: var(--shadow);
            z-index: 10;
        }

        .logo-badge {
            background: linear-gradient(135deg, var(--accent), var(--c-dataset));
            color: #fff;
            font-weight: 700;
            font-size: 11px;
            letter-spacing: 0.06em;
            padding: 3px 8px;
            border-radius: var(--radius);
            flex-shrink: 0;
        }

        .bundle-name {
            font-size: 15px;
            font-weight: 600;
            color: var(--text);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            max-width: 200px;
        }

        .header-spacer { flex: 1; }

        .header-controls {
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .ctrl-label {
            font-size: 12px;
            color: var(--muted);
            white-space: nowrap;
        }

        select.ctrl-select,
        button.ctrl-btn {
            font-family: inherit;
            font-size: 12px;
            padding: 4px 8px;
            border: 1px solid var(--border);
            border-radius: var(--radius);
            background: var(--surface);
            color: var(--text);
            cursor: pointer;
            transition: border-color 0.15s, background 0.15s;
        }

        select.ctrl-select:hover, button.ctrl-btn:hover,
        select.ctrl-select:focus, button.ctrl-btn:focus {
            outline: none;
            border-color: var(--accent);
            background: var(--accent-bg);
        }

        button.icon-btn {
            background: none;
            border: 1px solid var(--border);
            color: var(--muted);
            cursor: pointer;
            padding: 5px;
            border-radius: var(--radius);
            display: flex;
            align-items: center;
            justify-content: center;
            transition: background 0.15s, color 0.15s, border-color 0.15s;
        }

        button.icon-btn:hover {
            background: var(--accent-bg);
            color: var(--accent);
            border-color: var(--accent);
        }

        /* ── App Shell ──────────────────────────────────────────── */
        .app {
            display: flex;
            flex: 1;
            overflow: hidden;
        }

        /* ── Sidebar ────────────────────────────────────────────── */
        aside {
            width: var(--sidebar-w);
            flex-shrink: 0;
            background: var(--surface);
            border-right: 1px solid var(--border);
            display: flex;
            flex-direction: column;
            overflow: hidden;
        }

        .sidebar-top {
            padding: 10px;
            border-bottom: 1px solid var(--border);
            display: flex;
            flex-direction: column;
            gap: 6px;
        }

        /* Search */
        .search-wrap { position: relative; }

        .search-icon {
            position: absolute;
            left: 8px;
            top: 50%;
            transform: translateY(-50%);
            color: var(--muted);
            pointer-events: none;
        }

        input#search {
            width: 100%;
            padding: 5px 8px 5px 28px;
            border: 1px solid var(--border);
            border-radius: var(--radius);
            background: var(--bg);
            color: var(--text);
            font-family: inherit;
            font-size: 13px;
            transition: border-color 0.15s;
        }

        input#search:focus {
            outline: none;
            border-color: var(--accent);
        }

        /* Type filter */
        select#filter-type {
            width: 100%;
            padding: 4px 8px;
            border: 1px solid var(--border);
            border-radius: var(--radius);
            background: var(--bg);
            color: var(--text);
            font-family: inherit;
            font-size: 12px;
            cursor: pointer;
            transition: border-color 0.15s;
        }

        select#filter-type:focus {
            outline: none;
            border-color: var(--accent);
        }

        /* Nav tree */
        nav#nav-tree {
            flex: 1;
            overflow-y: auto;
            padding: 8px;
        }

        .nav-group-title {
            display: flex;
            align-items: center;
            gap: 6px;
            font-size: 11px;
            font-weight: 700;
            text-transform: uppercase;
            letter-spacing: 0.06em;
            color: var(--muted);
            padding: 8px 4px 4px;
        }

        .nav-dot {
            width: 7px;
            height: 7px;
            border-radius: 50%;
            flex-shrink: 0;
        }

        .nav-list { list-style: none; margin-bottom: 4px; }

        .nav-item {
            display: flex;
            align-items: center;
            gap: 6px;
            padding: 4px 6px;
            border-radius: var(--radius);
            cursor: pointer;
            color: var(--muted);
            font-size: 13px;
            transition: background 0.1s, color 0.1s;
            user-select: none;
        }

        .nav-item:hover { background: var(--bg); color: var(--text); }

        .nav-item.active {
            background: var(--accent-bg);
            color: var(--accent);
            font-weight: 500;
        }

        .nav-item-label {
            flex: 1;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }

        .nav-tag {
            font-size: 10px;
            background: var(--border);
            color: var(--muted);
            border-radius: 10px;
            padding: 1px 5px;
            flex-shrink: 0;
        }

        .nav-empty {
            color: var(--muted);
            font-size: 12px;
            text-align: center;
            padding: 16px 8px;
        }

        /* Sidebar footer — log button */
        .sidebar-footer {
            padding: 8px 10px;
            border-top: 1px solid var(--border);
        }

        button.log-btn {
            width: 100%;
            display: flex;
            align-items: center;
            gap: 6px;
            padding: 6px 8px;
            background: none;
            border: 1px solid var(--border);
            border-radius: var(--radius);
            color: var(--muted);
            font-family: inherit;
            font-size: 12px;
            cursor: pointer;
            transition: background 0.15s, color 0.15s, border-color 0.15s;
        }

        button.log-btn:hover {
            background: var(--accent-bg);
            color: var(--accent);
            border-color: var(--accent);
        }

        /* ── Main (Graph + Detail) ──────────────────────────────── */
        .main {
            flex: 1;
            display: flex;
            overflow: hidden;
            min-width: 0;
        }

        /* ── Graph ──────────────────────────────────────────────── */
        #graph {
            flex: 1 1 60%;
            background: var(--bg);
            border-right: 1px solid var(--border);
            position: relative;
            min-width: 0;
        }

        .graph-hint {
            position: absolute;
            bottom: 12px;
            left: 50%;
            transform: translateX(-50%);
            font-size: 11px;
            color: var(--muted);
            background: var(--surface);
            border: 1px solid var(--border);
            padding: 4px 12px;
            border-radius: 20px;
            pointer-events: none;
            white-space: nowrap;
            box-shadow: var(--shadow);
        }

        /* ── Detail Panel ───────────────────────────────────────── */
        #detail {
            flex: 0 0 40%;
            overflow-y: auto;
            background: var(--surface);
            min-width: 280px;
        }

        #detail-empty {
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            height: 100%;
            gap: 12px;
            padding: 2rem;
            text-align: center;
            color: var(--muted);
        }

        #detail-empty .empty-icon { font-size: 2.5rem; }
        #detail-empty p { font-size: 13px; line-height: 1.55; }

        /* Log overlay inside detail panel */
        #log-overlay {
            display: none;
            padding: 20px 24px;
        }

        .log-overlay-header {
            display: flex;
            align-items: center;
            gap: 8px;
            margin-bottom: 14px;
        }

        .log-overlay-header h2 {
            font-size: 14px;
            font-weight: 600;
            flex: 1;
        }

        button.close-log {
            background: none;
            border: none;
            cursor: pointer;
            color: var(--muted);
            font-size: 20px;
            line-height: 1;
            padding: 0;
            display: flex;
            align-items: center;
        }

        button.close-log:hover { color: var(--text); }

        /* Concept article */
        article#detail-content {
            padding: 20px 24px;
        }

        /* ── Detail: Type chip + Header ─────────────────────────── */
        .detail-header { margin-bottom: 12px; }

        .type-chip {
            display: inline-block;
            padding: 2px 8px;
            border-radius: 10px;
            font-size: 11px;
            font-weight: 600;
            color: #fff;
            background: var(--c-default);
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 6px;
        }

        .detail-header h1 {
            font-size: 18px;
            font-weight: 600;
            line-height: 1.3;
            color: var(--text);
            margin-bottom: 3px;
        }

        .detail-id {
            font-family: 'Fira Code', ui-monospace, monospace;
            font-size: 11px;
            color: var(--muted);
        }

        /* ── Frontmatter dl — viz.html style ────────────────────── */
        dl.frontmatter {
            display: grid;
            grid-template-columns: 90px 1fr;
            row-gap: 5px;
            column-gap: 12px;
            margin: 10px 0 14px;
            font-size: 13px;
        }

        dl.frontmatter dt {
            color: var(--muted);
            font-weight: 500;
            align-self: start;
            padding-top: 1px;
        }

        dl.frontmatter dd { margin: 0; word-break: break-word; }

        dl.frontmatter a {
            color: var(--accent);
            word-break: break-all;
            font-size: 12px;
            font-family: 'Fira Code', ui-monospace, monospace;
        }

        .tag {
            display: inline-block;
            padding: 1px 6px;
            margin: 0 4px 3px 0;
            border-radius: 4px;
            background: var(--code-bg);
            color: var(--muted);
            font-size: 11px;
            border: 1px solid var(--border);
        }

        hr { border: none; border-top: 1px solid var(--border); margin: 14px 0; }

        /* ── Markdown Body ──────────────────────────────────────── */
        .markdown-body {
            font-size: 13px;
            line-height: 1.6;
            color: var(--text);
        }

        .markdown-body h1 { font-size: 16px; margin: 18px 0 6px; padding-bottom: 4px; border-bottom: 1px solid var(--border); }
        .markdown-body h2 { font-size: 14px; margin: 14px 0 4px; font-weight: 600; }
        .markdown-body h3 { font-size: 13px; margin: 12px 0 4px; font-weight: 600; }
        .markdown-body p  { margin: 6px 0; }

        .markdown-body a { color: var(--accent); }
        .markdown-body a.internal { cursor: pointer; text-decoration: underline; }

        .markdown-body code {
            background: var(--code-bg);
            padding: 1px 4px;
            border-radius: 3px;
            font-size: 12px;
            font-family: 'Fira Code', ui-monospace, monospace;
            border: 1px solid var(--border);
        }

        .markdown-body pre {
            background: var(--code-bg);
            padding: 10px 12px;
            border-radius: var(--radius);
            overflow-x: auto;
            font-size: 12px;
            border: 1px solid var(--border);
            margin: 8px 0;
        }

        .markdown-body pre code { background: transparent; border: none; padding: 0; }

        .markdown-body ul, .markdown-body ol { padding-left: 22px; margin: 6px 0; }
        .markdown-body li { margin: 2px 0; }

        .markdown-body blockquote {
            border-left: 3px solid var(--accent);
            padding-left: 10px;
            color: var(--muted);
            margin: 8px 0;
            font-style: italic;
        }

        .markdown-body blockquote.alert {
            font-style: normal;
            padding: 8px 12px;
            border-radius: 0 var(--radius) var(--radius) 0;
        }

        .markdown-body blockquote.alert-note      { border-left-color: #3b82f6; background: rgba(59,130,246,0.06); }
        .markdown-body blockquote.alert-tip       { border-left-color: #10b981; background: rgba(16,185,129,0.06); }
        .markdown-body blockquote.alert-important { border-left-color: #8b5cf6; background: rgba(139,92,246,0.06); }
        .markdown-body blockquote.alert-warning   { border-left-color: #f59e0b; background: rgba(245,158,11,0.06); }
        .markdown-body blockquote.alert-caution   { border-left-color: #ef4444; background: rgba(239,68,68,0.06); }

        .markdown-body table { border-collapse: collapse; margin: 8px 0; width: 100%; }
        .markdown-body th, .markdown-body td { border: 1px solid var(--border); padding: 4px 8px; font-size: 12px; }
        .markdown-body th { background: var(--code-bg); font-weight: 600; }

        /* ── Backlinks — viz.html "Cited by" ────────────────────── */
        #detail-backlinks {
            margin-top: 18px;
            padding-top: 14px;
            border-top: 1px solid var(--border);
        }

        #detail-backlinks h2 {
            font-size: 11px;
            font-weight: 700;
            text-transform: uppercase;
            letter-spacing: 0.06em;
            color: var(--muted);
            margin-bottom: 8px;
        }

        #detail-backlinks ul { list-style: none; display: flex; flex-direction: column; gap: 4px; }

        #detail-backlinks a {
            color: var(--accent);
            font-size: 13px;
            cursor: pointer;
            text-decoration: none;
        }

        #detail-backlinks a:hover { text-decoration: underline; }

        /* ── Citations ──────────────────────────────────────────── */
        .citations-section {
            margin-top: 16px;
            padding-top: 14px;
            border-top: 1px solid var(--border);
        }

        .citations-section h2 {
            font-size: 11px;
            font-weight: 700;
            text-transform: uppercase;
            letter-spacing: 0.06em;
            color: var(--muted);
            margin-bottom: 8px;
        }

        .citation-card {
            display: flex;
            gap: 8px;
            padding: 8px 10px;
            border: 1px solid var(--border);
            border-radius: var(--radius);
            margin-bottom: 6px;
            font-size: 12px;
            background: var(--bg);
        }

        .citation-num { font-weight: 700; color: var(--accent); flex-shrink: 0; }
        .citation-card a { color: var(--accent); }
        .citation-host { color: var(--muted); font-family: 'Fira Code', monospace; font-size: 11px; }

        /* ── Scrollbar ──────────────────────────────────────────── */
        ::-webkit-scrollbar { width: 5px; height: 5px; }
        ::-webkit-scrollbar-track { background: transparent; }
        ::-webkit-scrollbar-thumb { background: var(--border); border-radius: 3px; }
        ::-webkit-scrollbar-thumb:hover { background: var(--muted); }

        /* ── Responsive ─────────────────────────────────────────── */
        @media (max-width: 900px) {
            :root { --sidebar-w: 200px; }
            .bundle-name { max-width: 140px; }
        }

        @media (max-width: 680px) {
            aside { display: none; }
            #graph { flex: 1 1 50%; }
            #detail { flex: 0 0 50%; }
        }
    </style>
</head>
<body>

<!-- ── Header ─────────────────────────────────────────────────── -->
<header>
    <div class="logo-badge">OKF</div>
    <strong class="bundle-name" id="bundle-name"></strong>
    <div class="header-spacer"></div>
    <div class="header-controls">
        <span class="ctrl-label">Layout</span>
        <select class="ctrl-select" id="layout-select" title="Graph layout algorithm">
            <option value="cose">Force</option>
            <option value="concentric">Concentric</option>
            <option value="breadthfirst">Breadth-first</option>
            <option value="circle">Circle</option>
            <option value="grid">Grid</option>
        </select>
        <button class="ctrl-btn" id="reset-btn">Reset view</button>
        <button class="icon-btn" id="theme-btn" title="Toggle dark/light mode" aria-label="Toggle theme">
            <svg id="sun-icon" xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24"
                 fill="none" stroke="currentColor" stroke-width="2" style="display:none">
                <circle cx="12" cy="12" r="4"/>
                <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/>
            </svg>
            <svg id="moon-icon" xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24"
                 fill="none" stroke="currentColor" stroke-width="2">
                <path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"/>
            </svg>
        </button>
    </div>
</header>

<!-- ── App ────────────────────────────────────────────────────── -->
<div class="app">

    <!-- ── Sidebar ────────────────────────────────────────────── -->
    <aside>
        <div class="sidebar-top">
            <div class="search-wrap">
                <svg class="search-icon" xmlns="http://www.w3.org/2000/svg" width="13" height="13"
                     viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="11" cy="11" r="8"/>
                    <path d="m21 21-4.35-4.35"/>
                </svg>
                <input id="search" type="search" placeholder="Search concepts…" autocomplete="off">
            </div>
            <select id="filter-type" title="Filter by concept type">
                <option value="">All types</option>
            </select>
        </div>
        <nav id="nav-tree"></nav>
        <div class="sidebar-footer">
            <button class="log-btn" id="log-btn">
                <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24"
                     fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M12 8v4l3 3"/><circle cx="12" cy="12" r="10"/>
                </svg>
                Update Log
            </button>
        </div>
    </aside>

    <!-- ── Main ───────────────────────────────────────────────── -->
    <div class="main">

        <!-- Graph canvas -->
        <div id="graph">
            <div class="graph-hint" id="graph-hint">Click a node to view details</div>
        </div>

        <!-- Detail panel -->
        <section id="detail">

            <!-- Empty state -->
            <div id="detail-empty">
                <span class="empty-icon">📊</span>
                <p>Click a node in the graph<br>or select a concept from the sidebar.</p>
            </div>

            <!-- Log overlay -->
            <div id="log-overlay">
                <div class="log-overlay-header">
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24"
                         fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M12 8v4l3 3"/><circle cx="12" cy="12" r="10"/>
                    </svg>
                    <h2>Update Log</h2>
                    <button class="close-log" id="close-log" title="Close">&times;</button>
                </div>
                <div class="markdown-body" id="log-body"></div>
            </div>

            <!-- Concept article -->
            <article id="detail-content" hidden>
                <div class="detail-header">
                    <span class="type-chip" id="detail-type"></span>
                    <h1 id="detail-title"></h1>
                    <div class="detail-id" id="detail-id"></div>
                </div>

                <dl class="frontmatter">
                    <dt>Description</dt><dd id="detail-description"></dd>
                    <dt>Resource</dt>  <dd id="detail-resource"></dd>
                    <dt>Tags</dt>      <dd id="detail-tags"></dd>
                </dl>

                <hr>

                <div class="markdown-body" id="detail-body"></div>

                <section id="detail-backlinks" hidden>
                    <h2>Cited by</h2>
                    <ul id="backlinks-list"></ul>
                </section>

                <div id="detail-citations"></div>
            </article>

        </section>
    </div>
</div>

<!-- Bundle data (injected by okf doc) -->
<script id="okf-bundle-data" type="application/json">{{.BundleJSON}}</script>

<!-- App logic -->
<script>
'use strict';

/* ═══════════════════════════════════════════════════════════════
   Data
═══════════════════════════════════════════════════════════════ */
const bundle   = JSON.parse(document.getElementById('okf-bundle-data').textContent);
const concepts = bundle.concepts || {};
const links    = bundle.links    || [];

/* ═══════════════════════════════════════════════════════════════
   Helpers
═══════════════════════════════════════════════════════════════ */

/** XSS-safe HTML encoder for user-controlled strings placed in innerHTML. */
function esc(s) {
    if (s == null) return '';
    const d = document.createElement('div');
    d.textContent = String(s);
    return d.innerHTML;
}

/* ═══════════════════════════════════════════════════════════════
   Type → color mapping (matches viz.html)
═══════════════════════════════════════════════════════════════ */
const TYPE_COLORS = {
    'dataset':          '#8b5cf6',
    'bigquery dataset': '#8b5cf6',
    'table':            '#3b82f6',
    'bigquery table':   '#3b82f6',
    'spanner table':    '#3b82f6',
    'postgres table':   '#3b82f6',
    'metric':           '#10b981',
    'reference':        '#10b981',
    'playbook':         '#f59e0b',
};

function typeColor(type) {
    if (!type) return '#94a3b8';
    const lower = type.toLowerCase();
    for (const key in TYPE_COLORS) {
        if (lower.includes(key)) return TYPE_COLORS[key];
    }
    return '#94a3b8';
}

/* ═══════════════════════════════════════════════════════════════
   Pre-computed indices
═══════════════════════════════════════════════════════════════ */

// Backlinks: for each concept id, which other concepts link TO it?
const backlinksIndex = {};
links.forEach(function(lnk) {
    if (!backlinksIndex[lnk.to]) backlinksIndex[lnk.to] = [];
    backlinksIndex[lnk.to].push(lnk.from);
});

// Node degree (in + out) for sizing
const degreeIndex = {};
links.forEach(function(lnk) {
    degreeIndex[lnk.from] = (degreeIndex[lnk.from] || 0) + 1;
    degreeIndex[lnk.to]   = (degreeIndex[lnk.to]   || 0) + 1;
});

/* ═══════════════════════════════════════════════════════════════
   App State
═══════════════════════════════════════════════════════════════ */
let cy            = null;
let activeId      = null;
let darkMode      = false;
let filterType    = '';
let searchQuery   = '';

/* ═══════════════════════════════════════════════════════════════
   Title
═══════════════════════════════════════════════════════════════ */
const bundleTitle = bundle.title || 'OKF Bundle';
document.getElementById('bundle-name').textContent = bundleTitle;
document.title = bundleTitle + ' — OKF Documentation';

/* ═══════════════════════════════════════════════════════════════
   marked.js — GFM alert extension
═══════════════════════════════════════════════════════════════ */
marked.use({
    renderer: {
        blockquote: function(quote) {
            var m = quote.match(/\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]/i);
            if (m) {
                var t = m[1].toUpperCase();
                var clean = quote.replace(/\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*<br\s*\/?>?/i, '');
                return '<blockquote class="alert alert-' + t.toLowerCase() + '">' + clean + '</blockquote>';
            }
            return '<blockquote>' + quote + '</blockquote>';
        }
    }
});

/* ═══════════════════════════════════════════════════════════════
   Graph — Cytoscape.js
═══════════════════════════════════════════════════════════════ */

function buildElements(fType, sQuery) {
    var nodes = [];
    var edges = [];

    var q = (sQuery || '').toLowerCase().trim();

    Object.values(concepts).forEach(function(c) {
        var matchType = !fType || (c.type || '').toLowerCase() === fType.toLowerCase();
        var matchSearch = !q ||
            (c.title || '').toLowerCase().includes(q) ||
            c.id.toLowerCase().includes(q) ||
            (c.description || '').toLowerCase().includes(q) ||
            (c.tags || []).join(' ').toLowerCase().includes(q);

        if (matchType && matchSearch) {
            var deg  = degreeIndex[c.id] || 0;
            var size = 20 + Math.min(deg * 4, 40); // 20–60 px
            nodes.push({
                group: 'nodes',
                data: {
                    id:          c.id,
                    label:       c.title || c.id,
                    type:        c.type  || '',
                    description: c.description || '',
                    color:       typeColor(c.type),
                    size:        size,
                }
            });
        }
    });

    var visibleIds = new Set(nodes.map(function(n) { return n.data.id; }));

    links.forEach(function(lnk) {
        if (visibleIds.has(lnk.from) && visibleIds.has(lnk.to)) {
            edges.push({
                group: 'edges',
                data: {
                    id:     lnk.from + '__' + lnk.to,
                    source: lnk.from,
                    target: lnk.to,
                }
            });
        }
    });

    return nodes.concat(edges);
}

function graphStyle() {
    var fontColor = darkMode ? '#f1f5f9' : '#0f172a';
    var edgeColor = darkMode ? '#475569' : '#cbd5e1';
    var bgColor   = darkMode ? '#1e293b' : '#ffffff';
    return [
        {
            selector: 'node',
            style: {
                'background-color':  'data(color)',
                'label':             'data(label)',
                'width':             'data(size)',
                'height':            'data(size)',
                'font-size':         11,
                'font-family':       'Inter, system-ui, sans-serif',
                'color':             fontColor,
                'text-valign':       'bottom',
                'text-halign':       'center',
                'text-margin-y':     4,
                'text-max-width':    120,
                'text-wrap':         'ellipsis',
                'border-width':      2,
                'border-color':      bgColor,
            }
        },
        {
            selector: 'node:selected',
            style: {
                'border-color': '#2563eb',
                'border-width': 3,
            }
        },
        {
            selector: 'node.faded',
            style: { 'opacity': 0.2 }
        },
        {
            selector: 'edge',
            style: {
                'width':              1.5,
                'line-color':         edgeColor,
                'target-arrow-color': edgeColor,
                'target-arrow-shape': 'triangle',
                'curve-style':        'bezier',
                'opacity':            0.7,
            }
        },
        {
            selector: 'edge.faded',
            style: { 'opacity': 0.08 }
        },
    ];
}

function initGraph() {
    cy = cytoscape({
        container: document.getElementById('graph'),
        elements:  buildElements(filterType, searchQuery),
        style:     graphStyle(),
        layout:    { name: 'cose', animate: false, padding: 40, randomize: false },
        minZoom:   0.1,
        maxZoom:   5,
    });

    // Click node → load detail
    cy.on('tap', 'node', function(evt) {
        var id = evt.target.data('id');
        loadDetail(id);
        highlightNode(id);
    });

    // Click background → unfade all
    cy.on('tap', function(evt) {
        if (evt.target === cy) {
            cy.elements().removeClass('faded');
            cy.elements().unselect();
        }
    });
}

function highlightNode(id) {
    if (!cy) return;
    cy.elements().addClass('faded');
    var node = cy.getElementById(id);
    if (node && node.length) {
        var neighborhood = node.closedNeighborhood();
        neighborhood.removeClass('faded');
        cy.elements().unselect();
        node.select();
    }
}

function runLayout(layoutName) {
    if (!cy) return;
    cy.layout({ name: layoutName || 'cose', animate: true, animationDuration: 400, padding: 40 }).run();
}

function refreshGraph() {
    if (!cy) { initGraph(); return; }
    cy.elements().remove();
    cy.add(buildElements(filterType, searchQuery));
    cy.style(graphStyle());
    var name = document.getElementById('layout-select').value;
    cy.layout({ name: name, animate: false, padding: 40 }).run();
}

/* ═══════════════════════════════════════════════════════════════
   Sidebar — type filter select population
═══════════════════════════════════════════════════════════════ */
function populateTypeFilter() {
    var sel   = document.getElementById('filter-type');
    var types = Array.from(new Set(
        Object.values(concepts).map(function(c) { return c.type; }).filter(Boolean)
    )).sort();
    types.forEach(function(t) {
        var opt = document.createElement('option');
        opt.value = t;
        opt.textContent = t;
        sel.appendChild(opt);
    });
}

/* ═══════════════════════════════════════════════════════════════
   Sidebar — nav tree
═══════════════════════════════════════════════════════════════ */
function renderNav() {
    var tree = document.getElementById('nav-tree');
    tree.innerHTML = '';

    var q = (searchQuery || '').toLowerCase().trim();

    var filtered = Object.values(concepts).filter(function(c) {
        var matchType = !filterType || (c.type || '').toLowerCase() === filterType.toLowerCase();
        var matchQ = !q ||
            (c.title || '').toLowerCase().includes(q) ||
            c.id.toLowerCase().includes(q) ||
            (c.description || '').toLowerCase().includes(q) ||
            (c.tags || []).join(' ').toLowerCase().includes(q);
        return matchType && matchQ;
    });

    if (filtered.length === 0) {
        tree.innerHTML = '<div class="nav-empty">No concepts found</div>';
        return;
    }

    // Group by type
    var groups = {};
    filtered.forEach(function(c) {
        var t = c.type || 'Other';
        if (!groups[t]) groups[t] = [];
        groups[t].push(c);
    });

    Object.keys(groups).sort().forEach(function(type) {
        var color = typeColor(type);

        var groupTitle = document.createElement('div');
        groupTitle.className = 'nav-group-title';
        groupTitle.innerHTML =
            '<span class="nav-dot" style="background:' + color + '"></span>' +
            esc(type);
        tree.appendChild(groupTitle);

        var ul = document.createElement('ul');
        ul.className = 'nav-list';

        var sorted = groups[type].slice().sort(function(a, b) {
            return (a.title || a.id).localeCompare(b.title || b.id);
        });

        sorted.forEach(function(c) {
            var li   = document.createElement('li');
            var item = document.createElement('div');
            item.className = 'nav-item' + (c.id === activeId ? ' active' : '');
            item.dataset.id = c.id;

            var label = document.createElement('span');
            label.className = 'nav-item-label';
            label.textContent = c.title || c.id;
            item.appendChild(label);

            if (c.tags && c.tags[0]) {
                var tag = document.createElement('span');
                tag.className = 'nav-tag';
                tag.textContent = c.tags[0];
                item.appendChild(tag);
            }

            item.addEventListener('click', function() {
                loadDetail(c.id);
                if (cy) {
                    var node = cy.getElementById(c.id);
                    if (node && node.length) {
                        cy.animate({ center: { eles: node }, zoom: Math.max(cy.zoom(), 1.2) }, { duration: 300 });
                        highlightNode(c.id);
                    }
                }
            });

            li.appendChild(item);
            ul.appendChild(li);
        });

        tree.appendChild(ul);
    });
}

/* ═══════════════════════════════════════════════════════════════
   Detail panel — concept renderer
═══════════════════════════════════════════════════════════════ */
async function loadDetail(id) {
    activeId = id;
    var meta = concepts[id];
    if (!meta) return;

    // Hide log overlay and empty state; show article
    document.getElementById('log-overlay').style.display    = 'none';
    document.getElementById('detail-empty').style.display   = 'none';
    document.getElementById('detail-content').hidden        = false;
    document.getElementById('graph-hint').style.display     = 'none';

    // Header
    var typeEl = document.getElementById('detail-type');
    typeEl.textContent   = meta.type || 'Concept';
    typeEl.style.background = typeColor(meta.type);
    document.getElementById('detail-title').textContent = meta.title || id;
    document.getElementById('detail-id').textContent    = id;

    // Frontmatter: description
    document.getElementById('detail-description').textContent = meta.description || '—';

    // Frontmatter: resource
    var resEl = document.getElementById('detail-resource');
    if (meta.resource) {
        var isUrl = meta.resource.startsWith('http');
        resEl.innerHTML = isUrl
            ? '<a href="' + esc(meta.resource) + '" target="_blank" rel="noopener noreferrer">' + esc(meta.resource) + '</a>'
            : esc(meta.resource);
    } else {
        resEl.textContent = '—';
    }

    // Frontmatter: tags
    var tagsEl = document.getElementById('detail-tags');
    if (meta.tags && meta.tags.length) {
        tagsEl.innerHTML = meta.tags.map(function(t) {
            return '<span class="tag">' + esc(t) + '</span>';
        }).join('');
    } else {
        tagsEl.textContent = '—';
    }

    // Body: show loading placeholder, then fetch full concept
    var bodyEl = document.getElementById('detail-body');
    bodyEl.innerHTML = '<span style="color:var(--muted);font-size:12px">Loading…</span>';

    var concept = meta;
    try {
        var resp = await fetch('concepts/' + id + '.json');
        if (resp.ok) concept = await resp.json();
    } catch (_) { /* fall back to index entry */ }

    // Render markdown body
    try {
        bodyEl.innerHTML = marked.parse(concept.body || '*No content.*');
    } catch (e) {
        bodyEl.innerHTML = '<pre>' + esc(concept.body) + '</pre>';
    }

    // Rewrite internal markdown links to in-app navigation
    bodyEl.querySelectorAll('a[href]').forEach(function(a) {
        var href = a.getAttribute('href');
        if (!href || href.startsWith('http') || href.startsWith('mailto:') || href.startsWith('#')) return;
        var path = href.split('#')[0].replace(/\.md$/, '');
        var resolved = resolveConceptId(path, id);
        if (resolved && concepts[resolved]) {
            a.href = 'javascript:void(0)';
            a.classList.add('internal');
            a.addEventListener('click', (function(rid) {
                return function(e) { e.preventDefault(); loadDetail(rid); highlightNode(rid); };
            }(resolved)));
        }
    });

    // Backlinks (Cited by)
    var bl       = backlinksIndex[id] || [];
    var blSec    = document.getElementById('detail-backlinks');
    var blList   = document.getElementById('backlinks-list');
    if (bl.length) {
        blSec.hidden  = false;
        blList.innerHTML = bl.map(function(fromId) {
            var fc = concepts[fromId];
            return '<li><a href="javascript:void(0)" data-id="' + esc(fromId) + '">' +
                esc(fc ? (fc.title || fromId) : fromId) + '</a></li>';
        }).join('');
        blList.querySelectorAll('a[data-id]').forEach(function(a) {
            a.addEventListener('click', function() {
                var fid = a.dataset.id;
                loadDetail(fid);
                highlightNode(fid);
            });
        });
    } else {
        blSec.hidden = true;
    }

    // Citations
    var citEl = document.getElementById('detail-citations');
    if (concept.citations && concept.citations.length) {
        citEl.innerHTML =
            '<div class="citations-section">' +
            '<h2>Citations &amp; Sources</h2>' +
            concept.citations.map(function(c) {
                var isLocal = !c.uri.startsWith('http');
                var host = c.uri;
                try { if (!isLocal) host = new URL(c.uri).hostname; } catch(_) {}
                var target = isLocal ? '' : ' target="_blank" rel="noopener noreferrer"';
                return '<div class="citation-card">' +
                    '<span class="citation-num">[' + esc(c.number) + ']</span>' +
                    '<div>' +
                        '<div><a href="' + esc(c.uri) + '"' + target + '>' +
                            esc(c.title) + (isLocal ? '' : ' ↗') +
                        '</a></div>' +
                        '<div class="citation-host">' + esc(host) + '</div>' +
                    '</div>' +
                '</div>';
            }).join('') +
            '</div>';
    } else {
        citEl.innerHTML = '';
    }

    // Scroll detail panel to top
    document.getElementById('detail').scrollTop = 0;

    // Update nav active state
    document.querySelectorAll('.nav-item').forEach(function(el) {
        el.classList.toggle('active', el.dataset.id === id);
    });

    // Update hash without triggering hashchange re-render
    history.replaceState(null, '', '#/concept/' + id);
}

/**
 * resolveConceptId resolves a relative or bare path from a markdown link
 * to an absolute concept ID present in the bundle.
 */
function resolveConceptId(path, fromId) {
    if (concepts[path]) return path;
    // Relative resolution from fromId directory
    if (fromId) {
        var parts  = fromId.split('/');
        parts.pop();
        var hparts = path.split('/');
        var res    = parts.slice();
        for (var i = 0; i < hparts.length; i++) {
            var p = hparts[i];
            if (p === '..') { res.pop(); }
            else if (p && p !== '.') { res.push(p); }
        }
        var resolved = res.join('/');
        if (concepts[resolved]) return resolved;
    }
    // Suffix match fallback
    var norm = path.replace(/^\.\//, '');
    for (var cid in concepts) {
        if (cid === norm || cid.endsWith('/' + norm)) return cid;
    }
    return null;
}

/* ═══════════════════════════════════════════════════════════════
   Update Log overlay
═══════════════════════════════════════════════════════════════ */
function showLog() {
    document.getElementById('detail-empty').style.display  = 'none';
    document.getElementById('detail-content').hidden       = true;
    document.getElementById('graph-hint').style.display    = 'none';
    var overlay = document.getElementById('log-overlay');
    overlay.style.display = 'block';
    var logBody = document.getElementById('log-body');
    if (bundle.log) {
        try { logBody.innerHTML = marked.parse(bundle.log); }
        catch(e) { logBody.textContent = bundle.log; }
    } else {
        logBody.innerHTML = '<p style="color:var(--muted)">No update log available in this bundle.</p>';
    }
    document.getElementById('detail').scrollTop = 0;
}

document.getElementById('log-btn').addEventListener('click', showLog);

document.getElementById('close-log').addEventListener('click', function() {
    document.getElementById('log-overlay').style.display = 'none';
    if (activeId) {
        document.getElementById('detail-content').hidden = false;
    } else {
        document.getElementById('detail-empty').style.display = 'flex';
        document.getElementById('graph-hint').style.display   = '';
    }
});

/* ═══════════════════════════════════════════════════════════════
   Theme toggle
═══════════════════════════════════════════════════════════════ */
var sunIcon  = document.getElementById('sun-icon');
var moonIcon = document.getElementById('moon-icon');

function applyTheme(isDark) {
    darkMode = isDark;
    document.documentElement.classList.toggle('dark', isDark);
    sunIcon.style.display  = isDark ? 'block' : 'none';
    moonIcon.style.display = isDark ? 'none'  : 'block';
    if (cy) cy.style(graphStyle());
}

document.getElementById('theme-btn').addEventListener('click', function() {
    applyTheme(!darkMode);
});

/* ═══════════════════════════════════════════════════════════════
   Event wiring
═══════════════════════════════════════════════════════════════ */
document.getElementById('search').addEventListener('input', function(e) {
    searchQuery = e.target.value;
    renderNav();
    refreshGraph();
});

document.getElementById('filter-type').addEventListener('change', function(e) {
    filterType = e.target.value;
    renderNav();
    refreshGraph();
});

document.getElementById('layout-select').addEventListener('change', function(e) {
    runLayout(e.target.value);
});

document.getElementById('reset-btn').addEventListener('click', function() {
    if (cy) cy.fit(undefined, 40);
});

/* ═══════════════════════════════════════════════════════════════
   Hash router
═══════════════════════════════════════════════════════════════ */
function route() {
    var hash = window.location.hash;
    if (hash.startsWith('#/concept/')) {
        var id = decodeURIComponent(hash.slice('#/concept/'.length));
        if (concepts[id]) {
            loadDetail(id);
            highlightNode(id);
        }
    } else if (hash === '#/log') {
        showLog();
    }
}

window.addEventListener('hashchange', route);

/* ═══════════════════════════════════════════════════════════════
   Bootstrap
═══════════════════════════════════════════════════════════════ */

// Light theme by default; honour system preference on first visit
applyTheme(window.matchMedia('(prefers-color-scheme: dark)').matches);

populateTypeFilter();
renderNav();
initGraph();
route();
</script>
</body>
</html>
`
