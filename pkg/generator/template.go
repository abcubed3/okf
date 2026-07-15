// Package generator compiles an OKF bundle into an interactive static web application.
package generator

// HTMLTemplate is the single-page HTML, CSS, and JS application template for rendering
// the interactive documentation catalog. It includes search, type/tag filters, visual
// relationship graphs via vis.js, and support for theme toggle mode.
const HTMLTemplate = `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} — OKF Documentation</title>
    
    <!-- Google Fonts -->
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;500;600;700&family=Fira+Code:wght@400;500&display=swap" rel="stylesheet">
    
    <!-- Dependencies via CDN — version-pinned with SRI hashes for supply-chain security -->
    <script src="https://cdn.jsdelivr.net/npm/marked@4.3.0/marked.min.js"
            integrity="sha256-xoB1Zy2Xbkd3OQVguqESGUhVvUQEsTZH2khVquH5Ngw="
            crossorigin="anonymous"></script>
    <script src="https://cdn.jsdelivr.net/npm/vis-network@9.1.9/standalone/umd/vis-network.min.js"
            integrity="sha256-9T+DPdub+X7+hWuwY31P6I8545mZx+lKS4r8jeihouU="
            crossorigin="anonymous"></script>
    
    <style>
        :root {
            --bg-primary: #f8fafc;
            --bg-secondary: #ffffff;
            --bg-sidebar: #f1f5f9;
            --border-color: #e2e8f0;
            --text-primary: #0f172a;
            --text-secondary: #475569;
            --accent-color: #3b82f6;
            --accent-hover: #2563eb;
            --accent-light: #eff6ff;
            --code-bg: #f8fafc;
            --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
            --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
            --transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
            
            /* Type Colors */
            --color-dataset: #3b82f6;
            --color-table: #10b981;
            --color-metric: #f59e0b;
            --color-playbook: #8b5cf6;
            --color-default: #64748b;
        }

        .dark {
            --bg-primary: #0f172a;
            --bg-secondary: #1e293b;
            --bg-sidebar: #0f172a;
            --border-color: #334155;
            --text-primary: #f8fafc;
            --text-secondary: #94a3b8;
            --accent-color: #3b82f6;
            --accent-hover: #60a5fa;
            --accent-light: #1e293b;
            --code-bg: #0f172a;
            --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.5);
            --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.3), 0 2px 4px -1px rgba(0, 0, 0, 0.2);
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: 'Outfit', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background-color: var(--bg-primary);
            color: var(--text-primary);
            height: 100vh;
            overflow: hidden;
            display: flex;
            flex-direction: column;
            transition: var(--transition);
        }

        /* Layout */
        .app-container {
            display: flex;
            flex: 1;
            overflow: hidden;
            position: relative;
        }

        /* Top Header */
        header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 0.75rem 1.5rem;
            background-color: var(--bg-secondary);
            border-bottom: 1px solid var(--border-color);
            z-index: 10;
            box-shadow: var(--shadow-sm);
        }

        .logo-area {
            display: flex;
            align-items: center;
            gap: 0.75rem;
        }

        .logo-badge {
            background: linear-gradient(135deg, var(--accent-color), var(--color-playbook));
            color: white;
            font-weight: 700;
            padding: 0.25rem 0.6rem;
            border-radius: 6px;
            font-size: 0.85rem;
            letter-spacing: 0.05em;
        }

        .logo-title {
            font-size: 1.25rem;
            font-weight: 600;
            letter-spacing: -0.02em;
        }

        .header-controls {
            display: flex;
            align-items: center;
            gap: 1rem;
        }

        /* Theme Toggle */
        .btn-icon {
            background: none;
            border: 1px solid var(--border-color);
            color: var(--text-secondary);
            cursor: pointer;
            padding: 0.5rem;
            border-radius: 8px;
            display: flex;
            align-items: center;
            justify-content: center;
            transition: var(--transition);
        }

        .btn-icon:hover {
            background-color: var(--accent-light);
            color: var(--accent-color);
            border-color: var(--accent-color);
        }

        /* Sidebar styling */
        aside {
            width: 320px;
            background-color: var(--bg-sidebar);
            border-right: 1px solid var(--border-color);
            display: flex;
            flex-direction: column;
            flex-shrink: 0;
            overflow-y: auto;
        }

        .sidebar-section {
            padding: 1.25rem;
            border-bottom: 1px solid var(--border-color);
        }

        /* Search */
        .search-container {
            position: relative;
        }

        .search-input {
            width: 100%;
            padding: 0.6rem 1rem 0.6rem 2.25rem;
            border: 1px solid var(--border-color);
            border-radius: 8px;
            background-color: var(--bg-secondary);
            color: var(--text-primary);
            font-family: inherit;
            font-size: 0.9rem;
            transition: var(--transition);
        }

        .search-input:focus {
            outline: none;
            border-color: var(--accent-color);
            box-shadow: 0 0 0 2px var(--accent-light);
        }

        .search-icon {
            position: absolute;
            left: 0.75rem;
            top: 50%;
            transform: translateY(-50%);
            color: var(--text-secondary);
            pointer-events: none;
            width: 16px;
            height: 16px;
        }

        /* Navigation Groups */
        .nav-tree {
            padding: 1rem;
            display: flex;
            flex-direction: column;
            gap: 1.25rem;
        }

        .nav-group-title {
            font-size: 0.75rem;
            font-weight: 700;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            color: var(--text-secondary);
            margin-bottom: 0.5rem;
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .nav-group-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
        }

        .nav-list {
            list-style: none;
            display: flex;
            flex-direction: column;
            gap: 0.25rem;
        }

        .nav-item-link {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 0.5rem 0.75rem;
            border-radius: 6px;
            color: var(--text-secondary);
            text-decoration: none;
            font-size: 0.9rem;
            font-weight: 500;
            transition: var(--transition);
        }

        .nav-item-link:hover, .nav-item-link.active {
            background-color: var(--bg-secondary);
            color: var(--text-primary);
            box-shadow: var(--shadow-sm);
        }

        .nav-item-link.active {
            border-left: 3px solid var(--accent-color);
            padding-left: calc(0.75rem - 3px);
        }

        .nav-item-title {
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            max-width: 80%;
        }

        .nav-item-badge {
            font-size: 0.7rem;
            padding: 0.1rem 0.4rem;
            border-radius: 10px;
            background-color: var(--border-color);
            color: var(--text-secondary);
        }

        /* Main Workspace */
        main {
            flex: 1;
            display: flex;
            flex-direction: column;
            overflow: hidden;
            background-color: var(--bg-primary);
        }

        .tab-bar {
            display: flex;
            background-color: var(--bg-secondary);
            border-bottom: 1px solid var(--border-color);
            padding: 0 1rem;
        }

        .tab-button {
            padding: 0.85rem 1.25rem;
            border: none;
            background: none;
            color: var(--text-secondary);
            font-family: inherit;
            font-weight: 600;
            font-size: 0.9rem;
            cursor: pointer;
            border-bottom: 2px solid transparent;
            transition: var(--transition);
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .tab-button:hover {
            color: var(--text-primary);
        }

        .tab-button.active {
            color: var(--accent-color);
            border-bottom-color: var(--accent-color);
        }

        /* Tab Content panels */
        .tab-panel {
            flex: 1;
            overflow-y: auto;
            display: none;
            padding: 2rem;
            position: relative;
        }

        .tab-panel.active {
            display: block;
        }

        /* Content Render Styles */
        .concept-header {
            margin-bottom: 1.5rem;
            border-bottom: 1px solid var(--border-color);
            padding-bottom: 1.5rem;
        }

        .concept-title-row {
            display: flex;
            align-items: center;
            justify-content: space-between;
            flex-wrap: wrap;
            gap: 1rem;
            margin-bottom: 0.75rem;
        }

        .concept-title {
            font-size: 2.25rem;
            font-weight: 700;
            letter-spacing: -0.03em;
        }

        .concept-meta {
            display: flex;
            flex-wrap: wrap;
            gap: 0.75rem;
            align-items: center;
        }

        .badge-type {
            font-size: 0.75rem;
            font-weight: 600;
            padding: 0.25rem 0.75rem;
            border-radius: 9999px;
            color: white;
        }

        .badge-resource {
            background-color: var(--accent-light);
            color: var(--accent-color);
            border: 1px solid var(--border-color);
            font-family: 'Fira Code', monospace;
            font-size: 0.75rem;
            padding: 0.2rem 0.5rem;
            border-radius: 4px;
        }

        .tag-list {
            display: flex;
            gap: 0.4rem;
        }

        .tag-badge {
            font-size: 0.7rem;
            background-color: var(--border-color);
            color: var(--text-secondary);
            padding: 0.15rem 0.5rem;
            border-radius: 12px;
        }

        /* Markdown rendering overrides */
        .markdown-body {
            line-height: 1.625;
            font-size: 1.05rem;
            color: var(--text-primary);
        }

        .markdown-body h1, .markdown-body h2, .markdown-body h3 {
            margin-top: 1.5rem;
            margin-bottom: 0.75rem;
            font-weight: 600;
            letter-spacing: -0.01em;
        }

        .markdown-body h1 { font-size: 1.75rem; }
        .markdown-body h2 { font-size: 1.4rem; border-bottom: 1px solid var(--border-color); padding-bottom: 0.3rem; }
        .markdown-body h3 { font-size: 1.2rem; }

        .markdown-body p {
            margin-bottom: 1rem;
        }

        .markdown-body a {
            color: var(--accent-color);
            text-decoration: none;
            border-bottom: 1px dashed var(--accent-color);
            transition: var(--transition);
        }

        .markdown-body a:hover {
            color: var(--accent-hover);
            border-bottom-style: solid;
        }

        .markdown-body ul, .markdown-body ol {
            margin-bottom: 1rem;
            padding-left: 1.5rem;
        }

        .markdown-body li {
            margin-bottom: 0.25rem;
        }

        .markdown-body code {
            font-family: 'Fira Code', monospace;
            background-color: var(--code-bg);
            padding: 0.15rem 0.3rem;
            border-radius: 4px;
            font-size: 0.85em;
            border: 1px solid var(--border-color);
        }

        .markdown-body pre {
            background-color: var(--code-bg);
            padding: 1rem;
            border-radius: 8px;
            overflow-x: auto;
            border: 1px solid var(--border-color);
            margin-bottom: 1rem;
        }

        .markdown-body pre code {
            background-color: transparent;
            padding: 0;
            border: none;
            font-size: 0.9rem;
        }

        .markdown-body blockquote {
            border-left: 4px solid var(--accent-color);
            padding-left: 1rem;
            color: var(--text-secondary);
            font-style: italic;
            margin-bottom: 1rem;
        }

        .markdown-body table {
            width: 100%;
            border-collapse: collapse;
            margin-bottom: 1rem;
        }

        .markdown-body th, .markdown-body td {
            border: 1px solid var(--border-color);
            padding: 0.5rem 0.75rem;
            text-align: left;
        }

        .markdown-body th {
            background-color: var(--bg-sidebar);
            font-weight: 600;
        }

        /* Alert styling */
        .markdown-body blockquote.alert {
            border-left-width: 4px;
            padding: 0.75rem 1rem;
            margin-bottom: 1rem;
            font-style: normal;
            border-radius: 0 6px 6px 0;
        }
        
        .markdown-body blockquote.alert-note {
            border-left-color: #3b82f6;
            background-color: rgba(59, 130, 246, 0.05);
        }
        
        .markdown-body blockquote.alert-tip {
            border-left-color: #10b981;
            background-color: rgba(16, 185, 129, 0.05);
        }
        
        .markdown-body blockquote.alert-important {
            border-left-color: #8b5cf6;
            background-color: rgba(139, 92, 246, 0.05);
        }
        
        .markdown-body blockquote.alert-warning {
            border-left-color: #f59e0b;
            background-color: rgba(245, 158, 11, 0.05);
        }
        
        .markdown-body blockquote.alert-caution {
            border-left-color: #ef4444;
            background-color: rgba(239, 68, 68, 0.05);
        }

        /* Graph Panel */
        #graph-container {
            width: 100%;
            height: 100%;
            min-height: 500px;
            background-color: var(--bg-primary);
        }

        /* Hover Tooltip/Card */
        .hover-card {
            position: absolute;
            background-color: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 0.75rem 1rem;
            max-width: 280px;
            box-shadow: var(--shadow-md);
            pointer-events: none;
            display: none;
            z-index: 1000;
            animation: fadeIn 0.15s ease-out;
        }

        .hover-card-title {
            font-weight: 600;
            font-size: 0.95rem;
            margin-bottom: 0.25rem;
        }

        .hover-card-meta {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            margin-bottom: 0.5rem;
        }

        .hover-card-desc {
            font-size: 0.8rem;
            color: var(--text-secondary);
            line-height: 1.4;
        }

        /* Empty State */
        .empty-state {
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            height: 100%;
            color: var(--text-secondary);
            text-align: center;
            gap: 1rem;
        }

        .empty-state-icon {
            font-size: 3rem;
        }

        /* Responsive adjustments */
        @media (max-width: 768px) {
            .app-container {
                flex-direction: column;
            }
            aside {
                width: 100%;
                height: 250px;
                border-right: none;
                border-bottom: 1px solid var(--border-color);
            }
        }

        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(4px); }
            to { opacity: 1; transform: translateY(0); }
        }

        @keyframes spin {
            from { transform: rotate(0deg); }
            to { transform: rotate(360deg); }
        }
    </style>
</head>
<body>

    <!-- Header -->
    <header>
        <div class="logo-area">
            <div class="logo-badge">OKF</div>
            <div class="logo-title" id="bundle-title">Data Knowledge Catalog</div>
        </div>
        
        <div class="header-controls">
            <!-- Theme Toggle -->
            <button class="btn-icon" id="theme-toggle" title="Toggle Dark/Light Mode">
                <svg id="sun-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="display:none;"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/></svg>
                <svg id="moon-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"/></svg>
            </button>
        </div>
    </header>

    <!-- App Container -->
    <div class="app-container">
        
        <!-- Sidebar -->
        <aside>
            <!-- Search Bar -->
            <div class="sidebar-section">
                <div class="search-container">
                    <svg class="search-icon" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <circle cx="11" cy="11" r="8"></circle>
                        <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
                    </svg>
                    <input type="text" class="search-input" id="search-bar" placeholder="Search concepts...">
                </div>
            </div>
            
            <!-- Navigation Links -->
            <nav class="nav-tree" id="navigation-tree">
                <!-- Dynamically Populated -->
            </nav>
        </aside>
        
        <!-- Main Panel -->
        <main>
            <!-- Tabs -->
            <div class="tab-bar">
                <button class="tab-button active" id="tab-doc-btn" onclick="switchTab('doc')">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1-2.5-2.5Z"/><path d="M6 6h10"/><path d="M6 10h10"/></svg>
                    Document
                </button>
                <button class="tab-button" id="tab-graph-btn" onclick="switchTab('graph')">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M3 20a6 6 0 0 1 12-1.2"/><path d="M18 20a6 6 0 0 1-6-6"/><circle cx="6" cy="6" r="3"/><path d="M6 9v5"/><path d="M9 6h5"/></svg>
                    Relationship Graph
                </button>
                <button class="tab-button" id="tab-log-btn" onclick="switchTab('log')">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 8v4l3 3"/><circle cx="12" cy="12" r="10"/></svg>
                    Update Log
                </button>
            </div>
            
            <!-- Document View Panel -->
            <div class="tab-panel active" id="panel-doc">
                <div id="concept-view-content">
                    <div class="empty-state">
                        <div class="empty-state-icon">📚</div>
                        <h2>No Concept Selected</h2>
                        <p>Select a concept from the sidebar or search to get started.</p>
                    </div>
                </div>
            </div>
            
            <!-- Graph View Panel -->
            <div class="tab-panel" id="panel-graph" style="padding: 0;">
                <div id="graph-container"></div>
            </div>

            <!-- Log View Panel -->
            <div class="tab-panel" id="panel-log">
                <div class="markdown-body" id="log-content">
                    <!-- Loaded dynamically -->
                </div>
            </div>
        </main>
    </div>

    <!-- Floating Hover Tooltip -->
    <div class="hover-card" id="hover-tooltip">
        <div class="hover-card-title" id="tooltip-title">Concept Title</div>
        <div class="hover-card-meta">
            <span class="badge-type" id="tooltip-badge-type" style="background-color: var(--color-default)">Type</span>
            <span class="hover-card-desc" id="tooltip-id">tables/users</span>
        </div>
        <div class="hover-card-desc" id="tooltip-desc">Description of the target concept goes here.</div>
    </div>

    <!-- Data Injection -->
    <script id="okf-bundle-data" type="application/json">
        {{.BundleJSON}}
    </script>

    <!-- App Logic -->
    <script>
        // Parse the injected bundle data
        const bundle = JSON.parse(document.getElementById('okf-bundle-data').textContent);
        
        // State management
        let activeConceptId = '';
        let visNetwork = null;

        // Color mapping by frontmatter type
        const typeColors = {
            'dataset': 'var(--color-dataset)',
            'table': 'var(--color-table)',
            'bigquery table': 'var(--color-table)',
            'spanner table': 'var(--color-table)',
            'postgres table': 'var(--color-table)',
            'metric': 'var(--color-metric)',
            'playbook': 'var(--color-playbook)'
        };

        function getTypeColor(type) {
            if (!type) return 'var(--color-default)';
            const lower = type.toLowerCase();
            for (const key in typeColors) {
                if (lower.includes(key)) {
                    return typeColors[key];
                }
            }
            return 'var(--color-default)';
        }

        // Configure marked.js alerts parser (GFM style alerts)
        const originalBlockquote = marked.Renderer.prototype.blockquote;
        marked.use({
            renderer: {
                blockquote(quote) {
                    const alertMatch = quote.match(/\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]/i);
                    if (alertMatch) {
                        const alertType = alertMatch[1].toUpperCase();
                        const cleanQuote = quote.replace(/\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*<br\s*\/?>?/i, '');
                        return '\n<blockquote class="alert alert-' + alertType.toLowerCase() + '">' + cleanQuote + '</blockquote>\n';
                    }
                    return '<blockquote>' + quote + '</blockquote>';
                }
            }
        });

        // Init Title
        if (bundle.title) {
            document.getElementById('bundle-title').textContent = bundle.title;
            document.title = bundle.title + ' — OKF Documentation';
        }

        // Setup theme toggle
        const themeToggle = document.getElementById('theme-toggle');
        const sunIcon = document.getElementById('sun-icon');
        const moonIcon = document.getElementById('moon-icon');

        themeToggle.addEventListener('click', () => {
            const isDark = document.documentElement.classList.toggle('dark');
            sunIcon.style.display = isDark ? 'none' : 'block';
            moonIcon.style.display = isDark ? 'block' : 'none';
            if (visNetwork) {
                // Re-render network graph with dark/light variables
                buildGraph();
            }
        });

        // Initialize theme UI
        const isSystemDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        if (isSystemDark) {
            document.documentElement.classList.add('dark');
            sunIcon.style.display = 'none';
            moonIcon.style.display = 'block';
        } else {
            document.documentElement.classList.remove('dark');
            sunIcon.style.display = 'block';
            moonIcon.style.display = 'none';
        }

        // Populate Navigation Tree
        function renderNavigation(filteredConcepts = null) {
            const conceptsToRender = filteredConcepts || Object.values(bundle.concepts);
            const navTree = document.getElementById('navigation-tree');
            navTree.innerHTML = '';

            if (conceptsToRender.length === 0) {
                navTree.innerHTML = '<div style="color: var(--text-secondary); text-align: center; font-size: 0.9rem; padding: 1rem;">No concepts found</div>';
                return;
            }

            // Group by concept type
            const groups = {};
            conceptsToRender.forEach(c => {
                const type = c.type || 'Other';
                if (!groups[type]) {
                    groups[type] = [];
                }
                groups[type].push(c);
            });

            // Sort group names
            const sortedGroups = Object.keys(groups).sort();

            sortedGroups.forEach(type => {
                const section = document.createElement('div');
                
                const title = document.createElement('div');
                title.className = 'nav-group-title';
                const dotColor = getTypeColor(type);
                title.innerHTML = '<span class="nav-group-dot" style="background-color: ' + dotColor + '"></span>' + type;
                section.appendChild(title);

                const ul = document.createElement('ul');
                ul.className = 'nav-list';

                // Sort items by Title/ID
                const sortedItems = groups[type].sort((a, b) => {
                    const titleA = a.title || a.id;
                    const titleB = b.title || b.id;
                    return titleA.localeCompare(titleB);
                });

                sortedItems.forEach(c => {
                    const li = document.createElement('li');
                    const link = document.createElement('a');
                    link.className = 'nav-item-link';
                    if (c.id === activeConceptId) {
                        link.classList.add('active');
                    }
                    link.href = '#/concept/' + c.id;
                    
                    const titleSpan = document.createElement('span');
                    titleSpan.className = 'nav-item-title';
                    titleSpan.textContent = c.title || c.id;
                    link.appendChild(titleSpan);

                    if (c.tags && c.tags.length > 0) {
                        const tagBadge = document.createElement('span');
                        tagBadge.className = 'nav-item-badge';
                        tagBadge.textContent = c.tags[0];
                        link.appendChild(tagBadge);
                    }

                    li.appendChild(link);
                    ul.appendChild(li);
                });

                section.appendChild(ul);
                navTree.appendChild(section);
            });
        }

        // Live Search
        const searchBar = document.getElementById('search-bar');
        searchBar.addEventListener('input', (e) => {
            const query = e.target.value.toLowerCase().trim();
            if (!query) {
                renderNavigation();
                return;
            }

            const filtered = Object.values(bundle.concepts).filter(c => {
                const title = (c.title || '').toLowerCase();
                const id = c.id.toLowerCase();
                const desc = (c.description || '').toLowerCase();
                const body = (c.body || '').toLowerCase();
                const tags = (c.tags || []).map(t => t.toLowerCase()).join(' ');

                return title.includes(query) ||
                       id.includes(query) ||
                       desc.includes(query) ||
                       body.includes(query) ||
                       tags.includes(query);
            });

            renderNavigation(filtered);
        });

        // Tab Switcher
        function switchTab(tabName) {
            // Deactivate all
            document.querySelectorAll('.tab-button').forEach(btn => btn.classList.remove('active'));
            document.querySelectorAll('.tab-panel').forEach(panel => panel.classList.remove('active'));

            // Activate target
            document.getElementById('tab-' + tabName + '-btn').classList.add('active');
            document.getElementById('panel-' + tabName).classList.add('active');

            if (tabName === 'graph') {
                // Initialize/Resize graph
                buildGraph();
            }
        }

        // Hover Card Tooltip Logic
        const tooltip = document.getElementById('hover-tooltip');
        function setupTooltips() {
            const markdownContainer = document.getElementById('concept-view-content');
            const links = markdownContainer.querySelectorAll('a');

            links.forEach(link => {
                const href = link.getAttribute('href');
                if (!href) return;

                // Check if this links to a concept
                const targetConcept = resolveConceptFromLink(href);
                if (targetConcept) {
                    // It's a local concept link. Rewrite it to point to the hash routing
                    link.href = '#/concept/' + targetConcept.id;

                    // Attach tooltip events
                    link.addEventListener('mouseenter', (e) => {
                        showTooltip(e, targetConcept);
                    });

                    link.addEventListener('mousemove', (e) => {
                        positionTooltip(e);
                    });

                    link.addEventListener('mouseleave', () => {
                        hideTooltip();
                    });
                }
            });
        }

        function resolveConceptFromLink(href) {
            // Handle cases: "../tables/users.md", "users.md", "tables/users"
            if (href.startsWith('http') || href.startsWith('mailto:') || href.startsWith('#')) {
                return null;
            }

            let path = href.split('#')[0]; // Strip anchor
            if (!path) return null;

            path = path.replace(/\.md$/, ''); // Strip md extension

            // Try direct lookup
            if (bundle.concepts[path]) {
                return bundle.concepts[path];
            }

            // Try relative resolution from activeConceptId
            if (activeConceptId) {
                const parts = activeConceptId.split('/');
                parts.pop(); // Remove active file
                
                // Process relative directories in link path (e.g. "../tables/users")
                const hrefParts = path.split('/');
                const resolvedParts = [...parts];
                
                for (const part of hrefParts) {
                    if (part === '..') {
                        resolvedParts.pop();
                    } else if (part !== '.' && part !== '') {
                        resolvedParts.push(part);
                    }
                }
                
                const resolvedId = resolvedParts.join('/');
                if (bundle.concepts[resolvedId]) {
                    return bundle.concepts[resolvedId];
                }
            }

            // Fallback: search by end-of-path match
            const normalizedPath = path.replace(/^\.\//, '');
            for (const id in bundle.concepts) {
                if (id === normalizedPath || id.endsWith('/' + normalizedPath)) {
                    return bundle.concepts[id];
                }
            }

            return null;
        }

        function showTooltip(e, concept) {
            document.getElementById('tooltip-title').textContent = concept.title || concept.id;
            document.getElementById('tooltip-id').textContent = concept.id;
            document.getElementById('tooltip-desc').textContent = concept.description || 'No description provided.';
            
            const badge = document.getElementById('tooltip-badge-type');
            badge.textContent = concept.type || 'Concept';
            badge.style.backgroundColor = getTypeColor(concept.type);

            tooltip.style.display = 'block';
            positionTooltip(e);
        }

        function positionTooltip(e) {
            // Read bounds
            const width = tooltip.offsetWidth;
            const height = tooltip.offsetHeight;
            
            let left = e.pageX + 15;
            let top = e.pageY + 15;

            // Prevent clipping screen edges
            if (left + width > window.innerWidth) {
                left = e.pageX - width - 15;
            }
            if (top + height > window.innerHeight) {
                top = e.pageY - height - 15;
            }

            tooltip.style.left = left + 'px';
            tooltip.style.top = top + 'px';
        }

        function hideTooltip() {
            tooltip.style.display = 'none';
        }

        // Render Concept — fetches body lazily from concepts/<id>.json
        async function renderConcept(id) {
            const meta = bundle.concepts[id];
            const contentPanel = document.getElementById('concept-view-content');

            if (!meta) {
                contentPanel.innerHTML = 
                    '<div class="empty-state">' +
                        '<div class="empty-state-icon">🔍</div>' +
                        '<h2>Concept Not Found</h2>' +
                        '<p>The concept ID "' + id + '" does not exist in this catalog bundle.</p>' +
                    '</div>';
                return;
            }

            activeConceptId = id;

            // Highlight in sidebar
            document.querySelectorAll('.nav-item-link').forEach(link => {
                if (link.getAttribute('href') === '#/concept/' + id) {
                    link.classList.add('active');
                } else {
                    link.classList.remove('active');
                }
            });

            // Show skeleton loader while fetching body
            contentPanel.innerHTML =
                '<div style="padding: 2rem; display: flex; align-items: center; gap: 1rem; color: var(--text-secondary);">' +
                    '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="animation: spin 1s linear infinite;"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>' +
                    '<span>Loading concept…</span>' +
                '</div>';

            // Fetch the per-concept JSON file
            let concept = meta;
            try {
                const resp = await fetch('concepts/' + id + '.json');
                if (resp.ok) {
                    const data = await resp.json();
                    concept = data;
                }
            } catch (fetchErr) {
                console.warn('Failed to fetch concept body for "' + id + '":', fetchErr);
            }

            // Build metadata header
            const typeColor = getTypeColor(concept.type);
            const resourceBlock = concept.resource 
                ? '<span class="badge-resource" title="Resource Path">' + concept.resource + '</span>'
                : '';
                
            const tagsBlock = (concept.tags && concept.tags.length > 0)
                ? '\n<div class="tag-list">\n' + 
                  concept.tags.map(t => '<span class="tag-badge">#' + t + '</span>').join('') + 
                  '\n</div>'
                : '';

            const timestampBlock = concept.timestamp
                ? '<span style="font-size: 0.8rem; color: var(--text-secondary)">Updated: ' + new Date(concept.timestamp).toLocaleDateString() + '</span>'
                : '';

            // Parse body markdown
            let htmlBody = '';
            try {
                htmlBody = marked.parse(concept.body || '');
            } catch (err) {
                htmlBody = '<p style="color: red">Error rendering markdown: ' + err.message + '</p><pre>' + concept.body + '</pre>';
            }

            contentPanel.innerHTML = 
                '<div class="concept-header">' +
                    '<div class="concept-title-row">' +
                        '<h1 class="concept-title">' + (concept.title || concept.id) + '</h1>' +
                        '<span class="badge-type" style="background-color: ' + typeColor + '">' + (concept.type || 'Concept') + '</span>' +
                    '</div>' +
                    '<div class="concept-meta">' +
                        resourceBlock +
                        tagsBlock +
                        timestampBlock +
                    '</div>' +
                '</div>' +
                '<div class="markdown-body">' +
                    htmlBody +
                '</div>';

            // Reset scroll
            document.getElementById('panel-doc').scrollTop = 0;

            // Setup custom links & tooltips
            setupTooltips();
        }

        // Build Relationship Graph
        function buildGraph() {
            const container = document.getElementById('graph-container');
            
            // Get colors based on dark/light mode
            const isDark = document.documentElement.classList.contains('dark');
            const fontColor = isDark ? '#f8fafc' : '#0f172a';
            const edgeColor = isDark ? '#475569' : '#cbd5e1';
            
            // Prepare nodes and edges
            const nodes = [];
            const edges = [];
            const edgeTracker = new Set();

            Object.values(bundle.concepts).forEach(c => {
                nodes.push({
                    id: c.id,
                    label: c.title || c.id,
                    title: c.description || c.id,
                    color: {
                        background: getTypeColor(c.type),
                        border: isDark ? '#0f172a' : '#ffffff',
                        highlight: {
                            background: '#3b82f6',
                            border: '#3b82f6'
                        }
                    },
                    shape: 'dot',
                    size: 15,
                    font: {
                        color: fontColor,
                        face: 'Outfit',
                        size: 12
                    }
                });

                // Find outgoing links in body to add edges
                const markdownLinkRegex = /\[[^\]]*\]\(([^)]+)\)/g;
                let match;
                while ((match = markdownLinkRegex.exec(c.body || '')) !== null) {
                    const targetUrl = match[1];
                    // Skip external or anchors
                    if (targetUrl.startsWith('http') || targetUrl.startsWith('mailto:') || targetUrl.startsWith('#')) {
                        continue;
                    }

                    // Resolve target concept
                    // Strip anchor
                    let targetPath = targetUrl.split('#')[0].replace(/\.md$/, '');
                    
                    // Try direct lookup or relative
                    let targetId = '';
                    if (bundle.concepts[targetPath]) {
                        targetId = targetPath;
                    } else {
                        // Attempt relative resolution
                        const parts = c.id.split('/');
                        parts.pop();
                        const hrefParts = targetPath.split('/');
                        const resolvedParts = [...parts];
                        for (const part of hrefParts) {
                            if (part === '..') {
                                resolvedParts.pop();
                            } else if (part !== '.' && part !== '') {
                                resolvedParts.push(part);
                            }
                        }
                        const resolvedId = resolvedParts.join('/');
                        if (bundle.concepts[resolvedId]) {
                            targetId = resolvedId;
                        }
                    }

                    if (targetId && targetId !== c.id) {
                        const edgeKey = c.id + '->' + targetId;
                        if (!edgeTracker.has(edgeKey)) {
                            edgeTracker.add(edgeKey);
                            edges.push({
                                from: c.id,
                                to: targetId,
                                arrows: 'to',
                                color: {
                                    color: edgeColor,
                                    highlight: '#3b82f6'
                                },
                                width: 1.5
                            });
                        }
                    }
                }
            });

            const data = {
                nodes: new vis.DataSet(nodes),
                edges: new vis.DataSet(edges)
            };

            const options = {
                physics: {
                    solver: 'forceAtlas2Based',
                    forceAtlas2Based: {
                        gravitationalConstant: -50,
                        centralGravity: 0.01,
                        springLength: 100,
                        springConstant: 0.08
                    }
                },
                interaction: {
                    hover: true,
                    tooltipDelay: 200
                }
            };

            if (visNetwork) {
                visNetwork.destroy();
            }

            visNetwork = new vis.Network(container, data, options);

            // Navigate to concept on node double-click or click
            visNetwork.on('click', function(params) {
                if (params.nodes.length > 0) {
                    const nodeId = params.nodes[0];
                    window.location.hash = '#/concept/' + nodeId;
                    switchTab('doc');
                }
            });
        }

        // Load Update Log
        function loadUpdateLog() {
            const logContentPanel = document.getElementById('log-content');
            if (bundle.log) {
                try {
                    logContentPanel.innerHTML = marked.parse(bundle.log);
                } catch (err) {
                    logContentPanel.innerHTML = '<p style="color: red">Error rendering log markdown: ' + err.message + '</p>';
                }
            } else {
                logContentPanel.innerHTML = 
                    '<div style="color: var(--text-secondary); text-align: center; padding: 2rem;">' +
                        '<h2>No Update Log Available</h2>' +
                        '<p>This bundle does not contain a log.md file.</p>' +
                    '</div>';
            }
        }

        // Router
        function route() {
            const hash = window.location.hash;
            
            if (hash.startsWith('#/concept/')) {
                const conceptId = hash.replace('#/concept/', '');
                renderConcept(conceptId);
                switchTab('doc');
            } else if (hash === '#/log') {
                switchTab('log');
            } else if (hash === '#/graph') {
                switchTab('graph');
            } else {
                // Default view: render first concept if available
                const conceptIds = Object.keys(bundle.concepts);
                if (conceptIds.length > 0) {
                    window.location.hash = '#/concept/' + conceptIds.sort()[0];
                } else {
                    // Empty bundle
                    document.getElementById('concept-view-content').innerHTML = 
                        '<div class="empty-state">' +
                            '<div class="empty-state-icon">📂</div>' +
                            '<h2>Catalog is Empty</h2>' +
                            '<p>No valid OKF concept documents found in this bundle.</p>' +
                        '</div>';
                }
            }
        }

        // Window Event Listeners
        window.addEventListener('hashchange', route);

        // Initial setup
        renderNavigation();
        loadUpdateLog();
        route();
    </script>
</body>
</html>
`
