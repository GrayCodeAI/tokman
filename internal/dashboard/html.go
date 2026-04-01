package dashboard

// dashboardHTML contains the embedded HTML template for the dashboard UI.
const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="icon" type="image/svg+xml" href="/logo">
    <title>TokMan Dashboard</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Fira+Code:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <script src="https://unpkg.com/lucide@latest"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: 'Fira Code', monospace;
            background: linear-gradient(135deg, #0f0f1a 0%, #1a1a2e 50%, #0d1b2a 100%);
            color: #f0f4f8;
            min-height: 100vh;
            padding: 2rem;
        }
        .container { max-width: 1600px; margin: 0 auto; }
        .header {
            text-align: center;
            margin-bottom: 2rem;
            padding: 1.5rem;
        }
        .logo {
            width: 80px;
            height: 80px;
            margin-bottom: 0.75rem;
            display: inline-block;
        }
        .logo img {
            width: 100%;
            height: 100%;
            object-fit: contain;
        }
        h1 { 
            font-size: 2rem;
            font-weight: 800;
            color: #22d3ee;
            letter-spacing: -0.02em;
        }
        .tagline {
            color: #8892b0;
            font-size: 0.9rem;
            margin-top: 0.25rem;
        }
        .toolbar {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 1.5rem;
            flex-wrap: wrap;
            gap: 1rem;
        }
        .time-selector {
            display: flex;
            gap: 0.5rem;
        }
        .pill-selector {
            display: flex;
            gap: 0.5rem;
            flex-wrap: wrap;
            margin-bottom: 1rem;
        }
        .time-btn {
            background: rgba(34, 211, 238, 0.1);
            border: 1px solid rgba(34, 211, 238, 0.3);
            color: #22d3ee;
            padding: 0.5rem 1rem;
            border-radius: 8px;
            cursor: pointer;
            font-size: 0.85rem;
            transition: all 0.2s;
        }
        .time-btn:hover, .time-btn.active {
            background: rgba(34, 211, 238, 0.2);
            border-color: #22d3ee;
        }
        .pill-btn {
            background: rgba(249, 115, 22, 0.08);
            border: 1px solid rgba(249, 115, 22, 0.25);
            color: #fb923c;
            padding: 0.4rem 0.85rem;
            border-radius: 999px;
            cursor: pointer;
            font-size: 0.8rem;
            transition: all 0.2s;
        }
        .pill-btn:hover, .pill-btn.active {
            background: rgba(249, 115, 22, 0.18);
            border-color: #fb923c;
        }
        .export-btn {
            background: rgba(16, 185, 129, 0.1);
            border: 1px solid rgba(16, 185, 129, 0.3);
            color: #10b981;
            padding: 0.5rem 1rem;
            border-radius: 8px;
            cursor: pointer;
            font-size: 0.85rem;
            text-decoration: none;
            transition: all 0.2s;
        }
        .export-btn:hover {
            background: rgba(16, 185, 129, 0.2);
        }
        .export-buttons {
            display: flex;
            gap: 0.5rem;
        }
        
        /* Alerts Banner */
        .alerts-banner {
            background: linear-gradient(145deg, rgba(245, 158, 11, 0.15) 0%, rgba(245, 158, 11, 0.05) 100%);
            border: 1px solid rgba(245, 158, 11, 0.4);
            border-radius: 12px;
            padding: 1rem 1.25rem;
            margin-bottom: 1.5rem;
            display: flex;
            align-items: center;
            gap: 0.75rem;
        }
        .alerts-banner.alert-info {
            background: linear-gradient(145deg, rgba(59, 130, 246, 0.15) 0%, rgba(59, 130, 246, 0.05) 100%);
            border-color: rgba(59, 130, 246, 0.4);
        }
        .alerts-banner.alert-warning {
            background: linear-gradient(145deg, rgba(245, 158, 11, 0.15) 0%, rgba(245, 158, 11, 0.05) 100%);
            border-color: rgba(245, 158, 11, 0.4);
        }
        .alerts-banner.alert-error {
            background: linear-gradient(145deg, rgba(239, 68, 68, 0.15) 0%, rgba(239, 68, 68, 0.05) 100%);
            border-color: rgba(239, 68, 68, 0.4);
        }
        .alerts-banner span {
            color: #f0f4f8;
            font-size: 0.9rem;
        }
        
        /* LLM Status Banner */
        .llm-banner {
            background: linear-gradient(145deg, rgba(34, 211, 238, 0.1) 0%, rgba(34, 211, 238, 0.03) 100%);
            border: 1px solid rgba(34, 211, 238, 0.25);
            border-radius: 16px;
            padding: 1.25rem;
            margin-bottom: 1.5rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
            flex-wrap: wrap;
            gap: 1rem;
        }
        .llm-info {
            display: flex;
            align-items: center;
            gap: 1rem;
        }
        .llm-icon {
            width: 40px;
            height: 40px;
            color: #22d3ee;
        }
        .llm-details h3 {
            color: #22d3ee;
            font-size: 1rem;
            margin-bottom: 0.25rem;
        }
        .llm-details .model {
            color: #8892b0;
            font-size: 0.85rem;
        }
        .llm-stats {
            display: flex;
            gap: 2rem;
        }
        .llm-stat {
            text-align: center;
        }
        .llm-stat .value {
            font-size: 1.5rem;
            font-weight: 700;
            color: #ffffff;
        }
        .llm-stat .label {
            font-size: 0.7rem;
            color: #8892b0;
            text-transform: uppercase;
        }
        
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
            gap: 1rem;
            margin-bottom: 1.5rem;
        }
        .stat-card {
            background: linear-gradient(145deg, rgba(255,255,255,0.08) 0%, rgba(255,255,255,0.02) 100%);
            border-radius: 16px;
            padding: 1.25rem;
            border: 1px solid rgba(255,255,255,0.1);
            transition: transform 0.2s ease, box-shadow 0.2s ease;
        }
        .stat-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 10px 30px rgba(255, 255, 255, 0.1);
        }
        .stat-card h3 {
            font-size: 0.75rem;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            margin-bottom: 0.5rem;
            font-weight: 600;
        }
        .stat-card .value {
            font-size: 1.75rem;
            font-weight: 700;
            color: #ffffff;
        }
        .stat-card .sub {
            font-size: 0.7rem;
            color: #8892b0;
            margin-top: 0.25rem;
        }
        /* Unique colors for each stat card */
        .stat-card.tokens {
            border-color: rgba(34, 211, 238, 0.4);
            background: linear-gradient(145deg, rgba(34, 211, 238, 0.15) 0%, rgba(34, 211, 238, 0.05) 100%);
        }
        .stat-card.tokens h3 { color: #22d3ee; }
        
        .stat-card.cost {
            border-color: rgba(16, 185, 129, 0.4);
            background: linear-gradient(145deg, rgba(16, 185, 129, 0.15) 0%, rgba(16, 185, 129, 0.05) 100%);
        }
        .stat-card.cost h3 { color: #10b981; }
        
        .stat-card.commands {
            border-color: rgba(59, 130, 246, 0.4);
            background: linear-gradient(145deg, rgba(59, 130, 246, 0.15) 0%, rgba(59, 130, 246, 0.05) 100%);
        }
        .stat-card.commands h3 { color: #3b82f6; }
        
        .stat-card.avg-savings {
            border-color: rgba(245, 158, 11, 0.4);
            background: linear-gradient(145deg, rgba(245, 158, 11, 0.15) 0%, rgba(245, 158, 11, 0.05) 100%);
        }
        .stat-card.avg-savings h3 { color: #f59e0b; }
        
        .stat-card.exec-time {
            border-color: rgba(236, 72, 153, 0.4);
            background: linear-gradient(145deg, rgba(236, 72, 153, 0.15) 0%, rgba(236, 72, 153, 0.05) 100%);
        }
        .stat-card.exec-time h3 { color: #ec4899; }

        .stat-card.context-reads {
            border-color: rgba(249, 115, 22, 0.4);
            background: linear-gradient(145deg, rgba(249, 115, 22, 0.15) 0%, rgba(249, 115, 22, 0.05) 100%);
        }
        .stat-card.context-reads h3 { color: #fb923c; }
        
        .stat-card.peak-hour {
            border-color: rgba(14, 165, 233, 0.4);
            background: linear-gradient(145deg, rgba(14, 165, 233, 0.15) 0%, rgba(14, 165, 233, 0.05) 100%);
        }
        .stat-card.peak-hour h3 { color: #0ea5e9; }
        
        .charts-grid {
            display: grid;
            grid-template-columns: 2fr 1fr;
            gap: 1.5rem;
            margin-bottom: 1.5rem;
        }
        @media (max-width: 900px) {
            .charts-grid { grid-template-columns: 1fr; }
        }
        .chart-container {
            background: linear-gradient(145deg, rgba(255,255,255,0.06) 0%, rgba(255,255,255,0.02) 100%);
            border-radius: 16px;
            padding: 1.5rem;
            border: 1px solid rgba(255,255,255,0.08);
        }
        .chart-container h2 {
            margin-bottom: 1rem;
            font-size: 1rem;
            font-weight: 600;
        }
        canvas { max-height: 250px; }
        
        /* Unique chart colors */
        .chart-savings {
            border-color: rgba(34, 211, 238, 0.3);
            background: linear-gradient(145deg, rgba(34, 211, 238, 0.08) 0%, rgba(34, 211, 238, 0.02) 100%);
        }
        .chart-savings h2 { color: #22d3ee; }
        
        .chart-hourly {
            border-color: rgba(6, 182, 212, 0.3);
            background: linear-gradient(145deg, rgba(6, 182, 212, 0.08) 0%, rgba(6, 182, 212, 0.02) 100%);
        }
        .chart-hourly h2 { color: #06b6d4; }
        
        .chart-commands {
            border-color: rgba(245, 158, 11, 0.3);
            background: linear-gradient(145deg, rgba(245, 158, 11, 0.08) 0%, rgba(245, 158, 11, 0.02) 100%);
        }
        .chart-commands h2 { color: #fbbf24; }
        
        .chart-composition {
            border-color: rgba(236, 72, 153, 0.3);
            background: linear-gradient(145deg, rgba(236, 72, 153, 0.08) 0%, rgba(236, 72, 153, 0.02) 100%);
        }
        .chart-composition h2 { color: #f472b6; }
        
        .chart-models {
            border-color: rgba(139, 92, 246, 0.3);
            background: linear-gradient(145deg, rgba(139, 92, 246, 0.08) 0%, rgba(139, 92, 246, 0.02) 100%);
        }
        .chart-models h2 { color: #a78bfa; }
        
        .chart-cache {
            border-color: rgba(16, 185, 129, 0.3);
            background: linear-gradient(145deg, rgba(16, 185, 129, 0.08) 0%, rgba(16, 185, 129, 0.02) 100%);
        }
        .chart-cache h2 { color: #10b981; }
        .chart-context {
            border-color: rgba(249, 115, 22, 0.3);
            background: linear-gradient(145deg, rgba(249, 115, 22, 0.08) 0%, rgba(249, 115, 22, 0.02) 100%);
        }
        .chart-context h2 { color: #fb923c; }
        
        .cache-stats {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 1rem;
            padding-top: 0.5rem;
        }
        .cache-stat {
            text-align: center;
            padding: 1rem;
            background: rgba(255,255,255,0.03);
            border-radius: 8px;
        }
        .cache-stat .value {
            font-size: 1.5rem;
            font-weight: 700;
            color: #10b981;
        }
        .cache-stat .label {
            font-size: 0.75rem;
            color: #8892b0;
            margin-top: 0.25rem;
        }
        .cache-stat.highlight .value {
            color: #22d3ee;
        }
        
        /* Daily Breakdown Table */
        .breakdown-section {
            margin-bottom: 1.5rem;
        }
        .breakdown-table {
            width: 100%;
            border-collapse: collapse;
            font-size: 0.85rem;
        }
        .breakdown-table th {
            text-align: left;
            padding: 0.75rem;
            color: #8892b0;
            font-weight: 600;
            border-bottom: 1px solid rgba(255,255,255,0.1);
        }
        .breakdown-table td {
            padding: 0.75rem;
            border-bottom: 1px solid rgba(255,255,255,0.05);
        }
        .breakdown-table tr:hover {
            background: rgba(255,255,255,0.02);
        }
        .breakdown-table .date { color: #22d3ee; }
        .breakdown-table .tokens { color: #10b981; font-weight: 600; }
        .breakdown-table .cost { color: #f59e0b; }
        
        .activity-section {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 1.5rem;
            margin-bottom: 1.5rem;
        }
        @media (max-width: 900px) {
            .activity-section { grid-template-columns: 1fr; }
        }
        .activity-list {
            background: linear-gradient(145deg, rgba(255,255,255,0.06) 0%, rgba(255,255,255,0.02) 100%);
            border-radius: 16px;
            padding: 1.5rem;
            border: 1px solid rgba(255,255,255,0.08);
        }
        .activity-list h2 {
            margin-bottom: 1rem;
            font-size: 1rem;
            font-weight: 600;
        }
        
        /* Unique activity section colors */
        .daily-section {
            border-color: rgba(34, 211, 238, 0.3);
            background: linear-gradient(145deg, rgba(34, 211, 238, 0.08) 0%, rgba(34, 211, 238, 0.02) 100%);
        }
        .daily-section h2 { color: #22d3ee; }
        
        .projects-section {
            border-color: rgba(16, 185, 129, 0.3);
            background: linear-gradient(145deg, rgba(16, 185, 129, 0.08) 0%, rgba(16, 185, 129, 0.02) 100%);
        }
        .projects-section h2 { color: #10b981; }
        
        .recent-section {
            border-color: rgba(59, 130, 246, 0.3);
            background: linear-gradient(145deg, rgba(59, 130, 246, 0.08) 0%, rgba(59, 130, 246, 0.02) 100%);
        }
        .recent-section h2 { color: #3b82f6; }
        
        .failures-section {
            border-color: rgba(239, 68, 68, 0.3);
            background: linear-gradient(145deg, rgba(239, 68, 68, 0.08) 0%, rgba(239, 68, 68, 0.02) 100%);
        }
        .failures-section h2 { color: #ef4444; }
        .activity-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 0.75rem 0;
            border-bottom: 1px solid rgba(255,255,255,0.05);
        }
        .activity-item:last-child { border-bottom: none; }
        .activity-item .cmd {
            font-family: monospace;
            font-size: 0.85rem;
            color: #22d3ee;
            max-width: 60%;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }
        .activity-item .meta {
            text-align: right;
            font-size: 0.75rem;
            color: #8892b0;
        }
        .activity-item .tokens {
            color: #10b981;
            font-weight: 600;
        }
        .failure-item {
            background: rgba(239, 68, 68, 0.1);
            border-radius: 8px;
            padding: 0.75rem;
            margin-bottom: 0.5rem;
        }
        .failure-item .cmd {
            color: #f87171;
        }
        .loading { text-align: center; padding: 2rem; opacity: 0.6; }
        .status-good { color: #10b981; }
        .status-warn { color: #f59e0b; }
        .status-bad { color: #ef4444; }
        .efficiency-bar {
            height: 8px;
            background: rgba(255,255,255,0.1);
            border-radius: 4px;
            margin-top: 0.5rem;
            overflow: hidden;
        }
        .efficiency-fill {
            height: 100%;
            background: linear-gradient(90deg, #22d3ee, #10b981);
            border-radius: 4px;
            transition: width 0.5s ease;
        }
        
        /* Project Stats */
        .project-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 0.5rem 0;
            border-bottom: 1px solid rgba(255,255,255,0.05);
        }
        .project-item .name {
            font-size: 0.8rem;
            color: #22d3ee;
            max-width: 50%;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }
        .project-item .stats {
            display: flex;
            gap: 1rem;
            font-size: 0.75rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo"><img src="/logo" alt="TokMan"></div>
            <h1>TokMan</h1>
            <p class="tagline">Token-Aware CLI Proxy • Real-time Analytics • LLM Integration</p>
        </div>
        
        <div class="toolbar">
            <div class="time-selector">
                <button class="time-btn active" data-days="7">7 Days</button>
                <button class="time-btn" data-days="14">14 Days</button>
                <button class="time-btn" data-days="30">30 Days</button>
            </div>
            <div class="export-buttons">
                <a href="/api/export/csv" class="export-btn" download><i data-lucide="file-text" style="width:14px;height:14px;vertical-align:middle;margin-right:4px"></i> Export CSV</a>
                <a href="/api/export/json" class="export-btn" download><i data-lucide="file-json" style="width:14px;height:14px;vertical-align:middle;margin-right:4px"></i> Export JSON</a>
                <a href="/api/report?type=weekly" class="export-btn" download><i data-lucide="file-bar-chart" style="width:14px;height:14px;vertical-align:middle;margin-right:4px"></i> Weekly Report</a>
            </div>
        </div>
        
        <!-- Alerts Banner -->
        <div class="alerts-banner" id="alerts-banner" style="display:none">
            <i data-lucide="alert-circle" style="width:20px;height:20px;color:#f59e0b"></i>
            <span id="alert-message"></span>
        </div>
        
        <!-- LLM Status Banner -->
        <div class="llm-banner" id="llm-banner">
            <div class="llm-info">
                <i data-lucide="bot" class="llm-icon"></i>
                <div class="llm-details">
                    <h3 id="llm-provider">Loading...</h3>
                    <div class="model" id="llm-model">Checking LLM status...</div>
                </div>
            </div>
            <div class="llm-stats" id="llm-stats">
                <!-- Filled by JS -->
            </div>
        </div>
        
        <div class="stats-grid">
            <div class="stat-card tokens">
                <h3>24h Tokens Saved</h3>
                <div class="value" id="tokens-saved-24h">--</div>
                <div class="sub">last 24 hours</div>
            </div>
            <div class="stat-card tokens-total">
                <h3>All-Time Savings</h3>
                <div class="value" id="tokens-saved-total">--</div>
                <div class="sub" id="efficiency-label">-- efficiency</div>
                <div class="efficiency-bar"><div class="efficiency-fill" id="efficiency-bar" style="width: 0%"></div></div>
            </div>
            <div class="stat-card cost">
                <h3>Est. Cost Savings</h3>
                <div class="value" id="cost-saved">$--</div>
                <div class="sub" id="cost-method">--</div>
            </div>
            <div class="stat-card commands">
                <h3>Commands Filtered</h3>
                <div class="value" id="commands-count">--</div>
                <div class="sub" id="cmd-rate">-- avg/day</div>
            </div>
            <div class="stat-card avg-savings">
                <h3>Avg Savings/Command</h3>
                <div class="value" id="avg-savings">--</div>
                <div class="sub">tokens</div>
            </div>
            <div class="stat-card exec-time">
                <h3>Avg Exec Time</h3>
                <div class="value" id="avg-exec">--</div>
                <div class="sub">ms avg</div>
            </div>
            <div class="stat-card context-reads">
                <h3>Smart Context Reads</h3>
                <div class="value" id="context-read-count">--</div>
                <div class="sub" id="context-read-saved">-- tokens saved</div>
            </div>
            <div class="stat-card context-reads">
                <h3>Bundle Vs Single</h3>
                <div class="value" id="context-bundle-saved">--</div>
                <div class="sub" id="context-single-saved">--</div>
            </div>
        </div>

        <div class="charts-grid">
            <div class="chart-container chart-savings">
                <h2><i data-lucide="trending-up" style="width:18px;height:18px;vertical-align:middle;margin-right:8px"></i>Token Savings Over Time</h2>
                <canvas id="savingsChart"></canvas>
            </div>
            <div class="chart-container chart-hourly">
                <h2><i data-lucide="clock" style="width:18px;height:18px;vertical-align:middle;margin-right:8px"></i>Hourly Distribution</h2>
                <canvas id="hourlyChart"></canvas>
            </div>
        </div>

        <div class="charts-grid">
            <div class="chart-container chart-commands">
                <h2><i data-lucide="award" style="width:18px;height:18px;vertical-align:middle;margin-right:8px"></i>Top Commands by Savings</h2>
                <canvas id="commandsChart"></canvas>
            </div>
            <div class="chart-container chart-composition">
                <h2><i data-lucide="pie-chart" style="width:18px;height:18px;vertical-align:middle;margin-right:8px"></i>Savings Composition</h2>
                <canvas id="compositionChart"></canvas>
            </div>
        </div>

        <!-- Model Breakdown & Cache Performance -->
        <div class="charts-grid">
            <div class="chart-container chart-models">
                <h2><i data-lucide="cpu" style="width:18px;height:18px;vertical-align:middle;margin-right:8px"></i>LLM Model Usage</h2>
                <canvas id="modelChart"></canvas>
            </div>
            <div class="chart-container chart-cache">
                <h2><i data-lucide="zap" style="width:18px;height:18px;vertical-align:middle;margin-right:8px"></i>Cache Performance</h2>
                <div id="cache-stats" class="cache-stats">
                    <div class="loading">Loading...</div>
                </div>
            </div>
        </div>

        <div class="charts-grid">
            <div class="chart-container chart-context">
                <h2><i data-lucide="route" style="width:18px;height:18px;vertical-align:middle;margin-right:8px"></i>Smart Read Trend</h2>
                <canvas id="contextTrendChart"></canvas>
            </div>
            <div class="activity-list projects-section">
                <h2><i data-lucide="file-stack" style="width:18px;height:18px;vertical-align:middle;margin-right:8px"></i>Top Smart Read Files</h2>
                <div id="context-read-top-files">
                    <div class="loading">Loading...</div>
                </div>
                <h2 style="margin-top:1.25rem"><i data-lucide="folders" style="width:18px;height:18px;vertical-align:middle;margin-right:8px"></i>Top Smart Read Projects</h2>
                <div id="context-read-projects">
                    <div class="loading">Loading...</div>
                </div>
                <h2 style="margin-top:1.25rem"><i data-lucide="sliders-horizontal" style="width:18px;height:18px;vertical-align:middle;margin-right:8px"></i>Context Mode Quality</h2>
                <div id="context-read-quality">
                    <div class="loading">Loading...</div>
                </div>
            </div>
        </div>

        <!-- Daily Breakdown Section -->
        <div class="activity-section breakdown-section">
            <div class="activity-list daily-section">
                <h2><i data-lucide="calendar" style="width:18px;height:18px;vertical-align:middle;margin-right:8px"></i>Daily Token Savings</h2>
                <table class="breakdown-table">
                    <thead>
                        <tr>
                            <th>Date</th>
                            <th>Saved</th>
                            <th>Commands</th>
                            <th>Efficiency</th>
                        </tr>
                    </thead>
                    <tbody id="daily-breakdown">
                        <tr><td colspan="4" class="loading">Loading...</td></tr>
                    </tbody>
                </table>
            </div>
            <div class="activity-list projects-section">
                <h2><i data-lucide="folder" style="width:18px;height:18px;vertical-align:middle;margin-right:8px"></i>Top Projects</h2>
                <div id="project-stats">
                    <div class="loading">Loading...</div>
                </div>
            </div>
        </div>

        <div class="activity-section">
            <div class="activity-list daily-section">
                <h2><i data-lucide="git-branch" style="width:18px;height:18px;vertical-align:middle;margin-right:8px"></i>Smart Read Activity</h2>
                <div class="pill-selector" id="context-read-filters">
                    <button class="pill-btn active" data-kind="all">All</button>
                    <button class="pill-btn" data-kind="read">CLI Read</button>
                    <button class="pill-btn" data-kind="delta">Delta</button>
                    <button class="pill-btn" data-kind="mcp">MCP</button>
                </div>
                <div id="context-read-list">
                    <div class="loading">Loading...</div>
                </div>
            </div>
        </div>

        <div class="activity-section">
            <div class="activity-list recent-section">
                <h2><i data-lucide="list" style="width:18px;height:18px;vertical-align:middle;margin-right:8px"></i>Recent Activity</h2>
                <div id="recent-list">
                    <div class="loading">Loading...</div>
                </div>
            </div>
            <div class="activity-list failures-section">
                <h2><i data-lucide="alert-triangle" style="width:18px;height:18px;vertical-align:middle;margin-right:8px"></i>Parse Failures <span id="failure-rate" style="font-size:0.8rem;color:#8892b0"></span></h2>
                <div id="failures-list">
                    <div class="loading">Loading...</div>
                </div>
            </div>
        </div>
    </div>

    <script>
        let currentDays = 7;
        let currentContextReadKind = 'all';
        let charts = {};
        
        async function fetchAPI(endpoint) {
            try {
                const response = await fetch(endpoint);
                return response.json();
            } catch (e) {
                console.error('API error:', e);
                return null;
            }
        }

        async function loadDashboard() {
            const contextReadEndpoint = currentContextReadKind === 'all'
                ? '/api/context-reads'
                : '/api/context-reads?kind=' + encodeURIComponent(currentContextReadKind);
            const [stats, economics, daily, hourly, recent, topCommands, failures, performance, llmStatus, dailyBreakdown, projectStats, alerts, modelBreakdown, cacheMetrics, contextReads, contextReadSummary, contextReadTrend, contextReadTopFiles, contextReadProjects, contextReadComparison, contextReadQuality] = await Promise.all([
                fetchAPI('/api/stats'),
                fetchAPI('/api/economics'),
                fetchAPI('/api/daily?days=' + currentDays),
                fetchAPI('/api/hourly'),
                fetchAPI('/api/recent'),
                fetchAPI('/api/top-commands'),
                fetchAPI('/api/failures'),
                fetchAPI('/api/performance'),
                fetchAPI('/api/llm-status'),
                fetchAPI('/api/daily-breakdown?days=' + currentDays),
                fetchAPI('/api/project-stats'),
                fetchAPI('/api/alerts'),
                fetchAPI('/api/model-breakdown'),
                fetchAPI('/api/cache-metrics'),
                fetchAPI(contextReadEndpoint),
                fetchAPI('/api/context-read-summary'),
                fetchAPI('/api/context-read-trend'),
                fetchAPI('/api/context-read-top-files'),
                fetchAPI('/api/context-read-projects'),
                fetchAPI('/api/context-read-comparison'),
                fetchAPI('/api/context-read-quality'),
            ]);

            // Update LLM Banner
            renderLLMStatus(llmStatus);
            
            // Render Alerts
            renderAlerts(alerts);
            
            // Render Model Breakdown
            renderModelChart(modelBreakdown);
            
            // Render Cache Metrics
            renderCacheMetrics(cacheMetrics);
            renderContextTrend(contextReadTrend || []);

            // Stats
            if (stats) {
                document.getElementById('tokens-saved-24h').textContent = (stats.tokens_saved_24h || 0).toLocaleString();
                document.getElementById('tokens-saved-total').textContent = (stats.tokens_saved_total || 0).toLocaleString();
                document.getElementById('commands-count').textContent = (stats.commands_count || 0).toLocaleString();
                const avgSavings = stats.commands_count > 0 ? Math.round(stats.tokens_saved_total / stats.commands_count) : 0;
                document.getElementById('avg-savings').textContent = avgSavings.toLocaleString();
                
                const efficiency = stats.original > 0 ? Math.round((stats.tokens_saved / stats.original) * 100) : 0;
                document.getElementById('efficiency-label').textContent = efficiency + '% reduction';
                document.getElementById('efficiency-bar').style.width = efficiency + '%';
                
                const avgPerDay = currentDays > 0 ? Math.round(stats.commands_count / currentDays) : 0;
                document.getElementById('cmd-rate').textContent = avgPerDay + ' avg/day';
                document.getElementById('context-read-count').textContent = (stats.context_read_commands || 0).toLocaleString();
                document.getElementById('context-read-saved').textContent = buildContextReadSummary(contextReadSummary, stats.context_read_saved || 0);
                renderContextReadComparison(contextReadComparison);
            }

            if (economics) {
                document.getElementById('cost-saved').textContent = '$' + (economics.estimated_cost || 0).toFixed(2);
                document.getElementById('cost-method').textContent = '@ $3/1M tokens';
            }

            if (performance) {
                document.getElementById('avg-exec').textContent = Math.round(performance.avg_exec_time_ms || 0);
            }

            // Peak hour
            if (hourly && hourly.length > 0) {
                let maxHour = 0, maxCount = 0;
                hourly.forEach(function(h) {
                    if (h.commands > maxCount) { maxCount = h.commands; maxHour = h.hour; }
                });
                var hourStr = maxHour > 12 ? (maxHour - 12) + ' PM' : maxHour === 12 ? '12 PM' : maxHour + ' AM';
            }

            // Charts
            renderSavingsChart(daily || []);
            renderHourlyChart(hourly || []);
            renderCommandsChart(topCommands || []);
            renderCompositionChart(topCommands || []);

            // Daily Breakdown
            renderDailyBreakdown(dailyBreakdown || []);
            
            // Project Stats
            renderProjectStats(projectStats || []);

            // Recent activity
            renderRecentActivity(recent || []);
            renderContextReads(contextReads || []);
            renderContextReadTopFiles(contextReadTopFiles || []);
            renderContextReadProjects(contextReadProjects || []);
            renderContextReadQuality(contextReadQuality);
            
            // Failures
            renderFailures(failures);
        }

        function renderAlerts(data) {
            var banner = document.getElementById('alerts-banner');
            var msgEl = document.getElementById('alert-message');
            
            if (!data || !data.enabled || !data.alerts || data.alerts.length === 0) {
                banner.style.display = 'none';
                return;
            }
            
            var alert = data.alerts[0];
            banner.style.display = 'flex';
            banner.className = 'alerts-banner alert-' + (alert.severity || 'warning');
            msgEl.textContent = alert.message;
        }

        function renderModelChart(data) {
            var ctx = document.getElementById('modelChart');
            if (!ctx) return;
            ctx = ctx.getContext('2d');
            if (charts.model) charts.model.destroy();
            
            if (!data || !data.available || !data.models || data.models.length === 0) {
                ctx.canvas.parentNode.innerHTML = '<div style="text-align:center;padding:2rem;color:#8892b0">No model data available<br><small style="color:#6b7280">Install ccusage for LLM tracking</small></div>';
                return;
            }
            
            charts.model = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: data.models.map(function(m) { return m.model; }),
                    datasets: [{
                        data: data.models.map(function(m) { return m.total_cost || 0; }),
                        backgroundColor: ['#a78bfa', '#8b5cf6', '#7c3aed', '#6d28d9', '#5b21b6'],
                        borderWidth: 0,
                    }]
                },
                options: {
                    responsive: true,
                    plugins: {
                        legend: { position: 'bottom', labels: { color: '#8892b0', font: { size: 10 } } },
                        tooltip: {
                            callbacks: {
                                label: function(context) {
                                    return context.label + ': $' + context.raw.toFixed(2);
                                }
                            }
                        }
                    }
                }
            });
        }

        function renderCacheMetrics(data) {
            var container = document.getElementById('cache-stats');
            if (!container) return;
            
            if (!data || data.error) {
                container.innerHTML = '<div style="text-align:center;padding:1rem;color:#8892b0">No cache data</div>';
                return;
            }
            
            var efficiency = (data.efficiency_pct || 0).toFixed(1);
            var savedCompact = formatCompact(data.total_saved || 0);
            var cacheReadCompact = data.cc_cache_read ? formatCompact(data.cc_cache_read) : 'N/A';
            var cacheCreateCompact = data.cc_cache_create ? formatCompact(data.cc_cache_create) : 'N/A';
            var contextHitRate = data.context_cache_hit_rate ? data.context_cache_hit_rate.toFixed(1) + '%' : '0.0%';
            var byKind = formatContextKindBreakdown(data.context_effectiveness_by_kind || []);
            var byProject = formatContextProjectBreakdown(data.context_effectiveness_by_project || []);
            
            container.innerHTML = 
                '<div class="cache-stat highlight">' +
                    '<div class="value">' + efficiency + '%</div>' +
                    '<div class="label">Efficiency</div>' +
                '</div>' +
                '<div class="cache-stat">' +
                    '<div class="value">' + savedCompact + '</div>' +
                    '<div class="label">Tokens Saved</div>' +
                '</div>' +
                '<div class="cache-stat">' +
                    '<div class="value">' + cacheReadCompact + '</div>' +
                    '<div class="label">Cache Reads</div>' +
                '</div>' +
                '<div class="cache-stat">' +
                    '<div class="value">' + cacheCreateCompact + '</div>' +
                    '<div class="label">Cache Creates</div>' +
                '</div>' +
                '<div class="cache-stat highlight">' +
                    '<div class="value">' + contextHitRate + '</div>' +
                    '<div class="label">Context Cache Hit Rate</div>' +
                '</div>' +
                '<div class="cache-stat" style="grid-column: span 2">' +
                    '<div class="value" style="font-size:1rem">' + byKind + '</div>' +
                    '<div class="label">Context Effectiveness By Kind</div>' +
                '</div>' +
                '<div class="cache-stat" style="grid-column: span 2">' +
                    '<div class="value" style="font-size:1rem">' + byProject + '</div>' +
                    '<div class="label">Top Context Projects</div>' +
                '</div>';
        }

        function formatContextKindBreakdown(items) {
            if (!items || items.length === 0) return 'No context data';
            return items.map(function(item) {
                return item.kind + ': ' + (item.tokens_saved || 0).toLocaleString();
            }).join(' · ');
        }

        function formatContextProjectBreakdown(items) {
            if (!items || items.length === 0) return 'No project data';
            return items.slice(0, 3).map(function(item) {
                var name = item.project ? item.project.split('/').pop() : 'unknown';
                return name + ': ' + (item.tokens_saved || 0).toLocaleString();
            }).join(' · ');
        }

        function renderLLMStatus(data) {
            if (!data) {
                document.getElementById('llm-provider').textContent = 'No LLM Data';
                document.getElementById('llm-model').textContent = 'Install ccusage for Claude Code integration';
                return;
            }
            
            document.getElementById('llm-provider').textContent = data.provider || 'Unknown';
            document.getElementById('llm-model').textContent = data.provider_model || 'Model info unavailable';
            
            const statsDiv = document.getElementById('llm-stats');
            if (data.ccusage_available && data.total_cost !== undefined) {
                statsDiv.innerHTML = 
                    '<div class="llm-stat">' +
                        '<div class="value">$' + (data.total_cost || 0).toFixed(2) + '</div>' +
                        '<div class="label">Total Spent</div>' +
                    '</div>' +
                    '<div class="llm-stat">' +
                        '<div class="value">' + formatCompact(data.total_input_tokens || 0) + '</div>' +
                        '<div class="label">Input Tokens</div>' +
                    '</div>' +
                    '<div class="llm-stat">' +
                        '<div class="value">' + formatCompact(data.total_output_tokens || 0) + '</div>' +
                        '<div class="label">Output Tokens</div>' +
                    '</div>' +
                    '<div class="llm-stat">' +
                        '<div class="value">' + formatCompact(data.total_cache_read || 0) + '</div>' +
                        '<div class="label">Cache Reads</div>' +
                    '</div>';
                
                // Update cost savings with real data if available
                if (data.total_input_tokens > 0) {
                    document.getElementById('cost-method').textContent = 'Real ccusage data';
                }
            } else {
                statsDiv.innerHTML = 
                    '<div class="llm-stat">' +
                        '<div class="value" style="color: #f59e0b">—</div>' +
                        '<div class="label">ccusage unavailable</div>' +
                    '</div>';
            }
        }

        function formatCompact(num) {
            if (num >= 1e6) return (num / 1e6).toFixed(1) + 'M';
            if (num >= 1e3) return (num / 1e3).toFixed(1) + 'K';
            return num.toString();
        }

        function renderDailyBreakdown(data) {
            const tbody = document.getElementById('daily-breakdown');
            if (!data || data.length === 0) {
                tbody.innerHTML = '<tr><td colspan="4" class="loading">No data available</td></tr>';
                return;
            }
            
            tbody.innerHTML = data.slice(0, 10).map(function(d) {
                return '<tr>' +
                    '<td class="date">' + (d.date || '--') + '</td>' +
                    '<td class="tokens">' + (d.tokens_saved || 0).toLocaleString() + '</td>' +
                    '<td>' + (d.commands || 0) + '</td>' +
                    '<td>' + (d.savings_pct || 0).toFixed(1) + '%</td>' +
                '</tr>';
            }).join('');
        }

        function renderProjectStats(data) {
            const container = document.getElementById('project-stats');
            if (!data || data.length === 0) {
                container.innerHTML = '<div class="activity-item"><span style="color:#8892b0">No project data</span></div>';
                return;
            }
            
            container.innerHTML = data.slice(0, 8).map(function(p) {
                const name = p.project ? p.project.split('/').pop() : 'unknown';
                return '<div class="project-item">' +
                    '<span class="name" title="' + (p.project || '') + '">' + name + '</span>' +
                    '<div class="stats">' +
                        '<span class="tokens">+' + (p.tokens_saved || 0).toLocaleString() + '</span>' +
                        '<span>' + (p.commands || 0) + ' cmds</span>' +
                    '</div>' +
                '</div>';
            }).join('');
        }

        function renderSavingsChart(data) {
            const ctx = document.getElementById('savingsChart').getContext('2d');
            if (charts.savings) charts.savings.destroy();
            
            const gradient = ctx.createLinearGradient(0, 0, 0, 250);
            gradient.addColorStop(0, 'rgba(34, 211, 238, 0.4)');
            gradient.addColorStop(1, 'rgba(34, 211, 238, 0.02)');
            
            charts.savings = new Chart(ctx, {
                type: 'line',
                data: {
                    labels: data.map(d => d.date ? d.date.slice(5) : ''),
                    datasets: [{
                        label: 'Tokens Saved',
                        data: data.map(d => d.tokens_saved || 0),
                        borderColor: '#22d3ee',
                        backgroundColor: gradient,
                        fill: true,
                        tension: 0.4,
                        pointBackgroundColor: '#22d3ee',
                        pointBorderColor: '#0f0f1a',
                        pointBorderWidth: 2,
                        pointRadius: 3,
                        pointHoverRadius: 5
                    }]
                },
                options: {
                    responsive: true,
                    plugins: { legend: { display: false } },
                    scales: {
                        y: { beginAtZero: true, grid: { color: 'rgba(255,255,255,0.05)' }, ticks: { color: '#8892b0' } },
                        x: { grid: { color: 'rgba(255,255,255,0.05)' }, ticks: { color: '#8892b0' } }
                    }
                }
            });
        }

        function renderHourlyChart(data) {
            const ctx = document.getElementById('hourlyChart').getContext('2d');
            if (charts.hourly) charts.hourly.destroy();
            
            charts.hourly = new Chart(ctx, {
                type: 'bar',
                data: {
                    labels: data.map(d => d.hour),
                    datasets: [{
                        label: 'Commands',
                        data: data.map(d => d.commands || 0),
                        backgroundColor: data.map(d => d.commands > 0 ? '#22d3ee' : 'rgba(34, 211, 238, 0.2)'),
                        borderRadius: 4,
                    }]
                },
                options: {
                    responsive: true,
                    plugins: { legend: { display: false } },
                    scales: {
                        y: { beginAtZero: true, grid: { color: 'rgba(255,255,255,0.05)' }, ticks: { color: '#8892b0' } },
                        x: { grid: { display: false }, ticks: { color: '#8892b0' } }
                    }
                }
            });
        }

        function renderCommandsChart(data) {
            const ctx = document.getElementById('commandsChart').getContext('2d');
            if (charts.commands) charts.commands.destroy();
            
            charts.commands = new Chart(ctx, {
                type: 'bar',
                data: {
                    labels: data.slice(0, 6).map(d => d.command || ''),
                    datasets: [{
                        label: 'Tokens Saved',
                        data: data.slice(0, 6).map(d => d.tokens_saved || 0),
                        backgroundColor: ['#22d3ee', '#06b6d4', '#0891b2', '#0e7490', '#155e75', '#164e63'],
                        borderRadius: 6,
                    }]
                },
                options: {
                    responsive: true,
                    indexAxis: 'y',
                    plugins: { legend: { display: false } },
                    scales: {
                        x: { beginAtZero: true, grid: { color: 'rgba(255,255,255,0.05)' }, ticks: { color: '#8892b0' } },
                        y: { grid: { display: false }, ticks: { color: '#e6f1ff' } }
                    }
                }
            });
        }

        function renderCompositionChart(data) {
            const ctx = document.getElementById('compositionChart').getContext('2d');
            if (charts.composition) charts.composition.destroy();
            
            charts.composition = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: data.slice(0, 5).map(d => d.command || ''),
                    datasets: [{
                        data: data.slice(0, 5).map(d => d.tokens_saved || 0),
                        backgroundColor: ['#22d3ee', '#06b6d4', '#0891b2', '#0e7490', '#155e75'],
                        borderWidth: 0,
                    }]
                },
                options: {
                    responsive: true,
                    plugins: {
                        legend: { position: 'bottom', labels: { color: '#8892b0', font: { size: 10 } } }
                    }
                }
            });
        }

        function renderRecentActivity(data) {
            const container = document.getElementById('recent-list');
            if (!data || data.length === 0) {
                container.innerHTML = '<div class="loading">No recent activity</div>';
                return;
            }
            container.innerHTML = data.slice(0, 8).map(item => 
                '<div class=\"activity-item\"><span class=\"cmd\">' + (item.command || 'unknown') + '</span>' +
                '<div class=\"meta\"><span class=\"tokens\">+' + (item.tokens_saved || 0).toLocaleString() + '</span>' +
                '<div>' + (item.exec_time_ms || 0) + 'ms</div></div></div>'
            ).join('');
        }

        function renderContextTrend(data) {
            const ctx = document.getElementById('contextTrendChart').getContext('2d');
            if (charts.contextTrend) charts.contextTrend.destroy();

            charts.contextTrend = new Chart(ctx, {
                type: 'line',
                data: {
                    labels: data.map(d => d.date ? d.date.slice(5) : ''),
                    datasets: [{
                        label: 'Smart Read Savings',
                        data: data.map(d => d.tokens_saved || 0),
                        borderColor: '#fb923c',
                        backgroundColor: 'rgba(249, 115, 22, 0.15)',
                        fill: true,
                        tension: 0.35,
                        pointRadius: 3
                    }]
                },
                options: {
                    responsive: true,
                    plugins: { legend: { display: false } },
                    scales: {
                        y: { beginAtZero: true, grid: { color: 'rgba(255,255,255,0.05)' }, ticks: { color: '#8892b0' } },
                        x: { grid: { color: 'rgba(255,255,255,0.05)' }, ticks: { color: '#8892b0' } }
                    }
                }
            });
        }

        function renderContextReads(data) {
            const container = document.getElementById('context-read-list');
            if (!data || data.length === 0) {
                container.innerHTML = '<div class="loading">No smart reads recorded yet</div>';
                return;
            }
            container.innerHTML = data.slice(0, 8).map(item =>
                '<div class=\"activity-item\"><span class=\"cmd\">' + (item.command || 'unknown') + '</span>' +
                '<div class=\"meta\"><span class=\"tokens\">+' + (item.tokens_saved || 0).toLocaleString() + '</span>' +
                '<div>' + ((item.reduction_pct || 0).toFixed(1)) + '% reduced</div></div></div>'
            ).join('');
        }

        function buildContextReadSummary(summary, fallbackSaved) {
            if (!summary) {
                return fallbackSaved.toLocaleString() + ' tokens saved';
            }

            const parts = [];
            ['read', 'delta', 'mcp'].forEach(function(kind) {
                if (!summary[kind] || !summary[kind].saved) return;
                parts.push(kind + ': ' + summary[kind].saved.toLocaleString());
            });
            if (parts.length === 0) {
                return fallbackSaved.toLocaleString() + ' tokens saved';
            }
            return parts.join(' · ');
        }

        function renderContextReadTopFiles(data) {
            const container = document.getElementById('context-read-top-files');
            if (!data || data.length === 0) {
                container.innerHTML = '<div class="activity-item"><span style="color:#8892b0">No smart read file data</span></div>';
                return;
            }
            container.innerHTML = data.slice(0, 5).map(item =>
                '<div class=\"project-item\"><span class=\"name\" title=\"' + (item.file || '') + '\">' + (item.file || 'unknown') + '</span>' +
                '<div class=\"stats\"><span class=\"tokens\">+' + (item.tokens_saved || 0).toLocaleString() + '</span><span>' + (item.commands || 0) + ' reads</span></div></div>'
            ).join('');
        }

        function renderContextReadProjects(data) {
            const container = document.getElementById('context-read-projects');
            if (!data || data.length === 0) {
                container.innerHTML = '<div class="activity-item"><span style="color:#8892b0">No smart read project data</span></div>';
                return;
            }
            container.innerHTML = data.slice(0, 5).map(item => {
                const name = item.project ? item.project.split('/').pop() : 'unknown';
                return '<div class=\"project-item\"><span class=\"name\" title=\"' + (item.project || '') + '\">' + name + '</span>' +
                    '<div class=\"stats\"><span class=\"tokens\">+' + (item.tokens_saved || 0).toLocaleString() + '</span><span>' + (item.commands || 0) + ' reads</span></div></div>';
            }).join('');
        }

        function renderContextReadComparison(data) {
            const bundle = data && data.bundle ? data.bundle : { tokens_saved: 0, commands: 0 };
            const single = data && data.single ? data.single : { tokens_saved: 0, commands: 0 };
            document.getElementById('context-bundle-saved').textContent = '+' + (bundle.tokens_saved || 0).toLocaleString();
            document.getElementById('context-single-saved').textContent =
                'single: +' + (single.tokens_saved || 0).toLocaleString() +
                ' · avg related: ' + ((bundle.avg_related_files || 0).toFixed ? bundle.avg_related_files.toFixed(1) : '0.0');
        }

        function renderContextReadQuality(data) {
            const container = document.getElementById('context-read-quality');
            const modes = data && data.modes ? data.modes : [];
            if (!modes || modes.length === 0) {
                container.innerHTML = '<div class="activity-item"><span style="color:#8892b0">No mode quality data yet</span></div>';
                return;
            }
            container.innerHTML = modes.slice(0, 5).map(item =>
                '<div class=\"project-item\"><span class=\"name\">' + (item.mode || 'unknown') + '</span>' +
                '<div class=\"stats\"><span class=\"tokens\">+' + Math.round(item.avg_saved_tokens || 0).toLocaleString() + '</span>' +
                '<span>' + (item.commands || 0) + ' reads</span><span>' + ((item.reduction_pct || 0).toFixed(1)) + '%</span></div></div>'
            ).join('');
        }

        function renderFailures(data) {
            const container = document.getElementById('failures-list');
            const rateEl = document.getElementById('failure-rate');
            
            if (!data || data.total_failures === 0) {
                container.innerHTML = '<div class="activity-item"><span style="color:#10b981">✓ No parse failures</span></div>';
                rateEl.textContent = '';
                return;
            }
            
            rateEl.textContent = '(' + (data.recovery_rate || 0).toFixed(0) + '% recovered)';
            
            const failures = data.top_failures || [];
            container.innerHTML = failures.slice(0, 5).map(f => 
                '<div class=\"failure-item\"><span class=\"cmd\">' + (f.command || 'unknown') + '</span>' +
                '<span style=\"color:#f87171;font-size:0.75rem\">' + (f.count || 0) + '×</span></div>'
            ).join('');
        }

        // Time selector
        document.querySelectorAll('.time-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.time-btn').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                currentDays = parseInt(btn.dataset.days);
                loadDashboard();
            });
        });

        document.querySelectorAll('.pill-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.pill-btn').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                currentContextReadKind = btn.dataset.kind;
                loadDashboard();
            });
        });

        loadDashboard();
        setInterval(loadDashboard, 30000);
        
        // Initialize Lucide icons
        lucide.createIcons();
    </script>
</body>
</html>`
