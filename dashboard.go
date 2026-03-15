package miru

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const dashboardMaxEntries = 500

type LogEntry struct {
	At   string `json:"at"`
	Tag  string `json:"tag"`
	Body string `json:"body"`
}

type dashboardHub struct {
	mu      sync.RWMutex
	entries []LogEntry
	clients map[chan []byte]struct{}
}

func newDashboardHub() *dashboardHub {
	return &dashboardHub{
		entries: make([]LogEntry, 0, dashboardMaxEntries),
		clients: make(map[chan []byte]struct{}),
	}
}

func (h *dashboardHub) Send(e LogEntry) {
	e.At = time.Now().Format("2006-01-02 15:04:05.000")

	data, err := json.Marshal(e)
	if err != nil {
		return
	}

	h.mu.Lock()

	h.entries = append(h.entries, e)
	if len(h.entries) > dashboardMaxEntries {
		h.entries = h.entries[len(h.entries)-dashboardMaxEntries:]
	}

	clients := make([]chan []byte, 0, len(h.clients))
	for ch := range h.clients {
		clients = append(clients, ch)
	}

	h.mu.Unlock()

	for _, ch := range clients {
		select {
		case ch <- data:
		default:
		}
	}
}

func (h *dashboardHub) Recent() []LogEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	out := make([]LogEntry, len(h.entries))
	copy(out, h.entries)

	return out
}

func (h *dashboardHub) Subscribe() ([]LogEntry, chan []byte) {
	h.mu.Lock()

	ch := make(chan []byte, 64)
	h.clients[ch] = struct{}{}

	recent := make([]LogEntry, len(h.entries))
	copy(recent, h.entries)

	h.mu.Unlock()

	return recent, ch
}

func (h *dashboardHub) Unsubscribe(ch chan []byte) {
	h.mu.Lock()
	delete(h.clients, ch)
	close(ch)
	h.mu.Unlock()
}

func (h *dashboardHub) RunServer(port int) *http.Server {
	if port <= 0 {
		port = 8765
	}

	addr := ":" + strconv.Itoa(port)

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(dashboardHTML)
	})

	mux.HandleFunc("/events", h.serveSSE)
	mux.HandleFunc("/api/recent", h.serveRecent)

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		fmt.Printf("[Miru Dashboard]: Broadcasting logs at http://localhost:%d\n", port)
		_ = srv.ListenAndServe()
	}()

	return srv
}

func (h *dashboardHub) serveRecent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	enc := json.NewEncoder(w)
	_ = enc.Encode(h.Recent())
}

func (h *dashboardHub) serveSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	recent, ch := h.Subscribe()
	defer h.Unsubscribe(ch)

	ctx := r.Context()

	for _, e := range recent {

		data, err := json.Marshal(e)
		if err != nil {
			continue
		}

		if _, err := w.Write([]byte("data: ")); err != nil {
			return
		}

		if _, err := w.Write(data); err != nil {
			return
		}

		if _, err := w.Write([]byte("\n\n")); err != nil {
			return
		}

		flusher.Flush()
	}

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {

		case <-ctx.Done():
			return

		case data := <-ch:

			if _, err := w.Write([]byte("data: ")); err != nil {
				return
			}

			if _, err := w.Write(data); err != nil {
				return
			}

			if _, err := w.Write([]byte("\n\n")); err != nil {
				return
			}

			flusher.Flush()

		case <-heartbeat.C:

			if _, err := w.Write([]byte(": heartbeat\n\n")); err != nil {
				return
			}

			flusher.Flush()
		}
	}
}

var dashboardHTML = []byte(`
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Miru Dashboard</title>
<script src="https://unpkg.com/vue@3/dist/vue.global.js"></script>
<style>

* { box-sizing: border-box; }

body {
	font-family: ui-monospace, 'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', 'Source Code Pro', monospace;
	font-size: 14px;
	margin: 0;
	background: linear-gradient(135deg, #0f0f0f 0%, #1a1a1a 100%);
	color: #e0e0e0;
	min-height: 100vh;
}

.header {
	padding: 16px 20px;
	border-bottom: 2px solid #2a2a2a;
	background: linear-gradient(90deg, #1a1a1a 0%, #2d2d2d 100%);
	box-shadow: 0 2px 10px rgba(0,0,0,0.3);
	position: sticky;
	top: 0;
	z-index: 100;
}

.header h1 {
	margin: 0;
	font-size: 1.5rem;
	font-weight: 700;
	color: #4fc3f7;
	text-shadow: 0 0 10px rgba(79, 195, 247, 0.3);
}

.header p {
	margin: 6px 0 0;
	font-size: 0.9rem;
	color: #9e9e9e;
}

.status-bar {
	padding: 12px 20px;
	background: #252525;
	border-bottom: 1px solid #2a2a2a;
	display: flex;
	justify-content: space-between;
	align-items: center;
	font-size: 0.85rem;
	flex-wrap: wrap;
	gap: 16px;
}

.status-indicator {
	display: flex;
	align-items: center;
	gap: 8px;
}

.status-dot {
	width: 8px;
	height: 8px;
	border-radius: 50%;
	background: #4caf50;
	animation: pulse 2s infinite;
}

@keyframes pulse {
	0%, 100% { opacity: 1; }
	50% { opacity: 0.5; }
}

.count-cards {
	display: flex;
	gap: 12px;
	flex-wrap: wrap;
}

.count-card {
	background: rgba(255,255,255,0.05);
	border: 1px solid rgba(255,255,255,0.1);
	border-radius: 8px;
	padding: 6px 10px;
	display: flex;
	align-items: center;
	gap: 6px;
	font-size: 0.75rem;
	transition: all 0.2s;
	cursor: pointer;
}

.count-card:hover {
	background: rgba(255,255,255,0.08);
	transform: translateY(-1px);
}

.count-card.active {
	background: rgba(79, 195, 247, 0.2);
	border-color: #4fc3f7;
}

.count-dot {
	width: 6px;
	height: 6px;
	border-radius: 50%;
}

.count-number {
	font-weight: 600;
	color: #fff;
}

.count-label {
	color: #ccc;
	margin-left: 2px;
}

.search-bar {
	padding: 12px 20px;
	background: #1a1a1a;
	border-bottom: 1px solid #2a2a2a;
	display: flex;
	gap: 12px;
	align-items: center;
}

.search-input {
	flex: 1;
	max-width: 400px;
	padding: 8px 12px;
	background: #2a2a2a;
	border: 1px solid #444;
	border-radius: 6px;
	color: #e0e0e0;
	font-size: 0.85rem;
	font-family: inherit;
}

.search-input:focus {
	outline: none;
	border-color: #4fc3f7;
}

.controls {
	display: flex;
	gap: 8px;
	align-items: center;
}

.control-btn {
	padding: 6px 12px;
	background: #333;
	border: 1px solid #444;
	border-radius: 6px;
	color: #ccc;
	font-size: 0.8rem;
	cursor: pointer;
	transition: all 0.2s;
	font-family: inherit;
}

.control-btn:hover {
	background: #444;
	border-color: #666;
}

.control-btn.active {
	background: #4fc3f7;
	border-color: #4fc3f7;
	color: #000;
}

.control-btn.danger {
	background: #ff5252;
	border-color: #ff5252;
	color: #fff;
}

.control-btn.danger:hover {
	background: #ff3838;
	border-color: #ff3838;
}

.shortcuts-help {
	position: fixed;
	top: 80px;
	right: 20px;
	background: #252525;
	border: 1px solid #444;
	border-radius: 8px;
	padding: 12px;
	font-size: 0.75rem;
	color: #ccc;
	z-index: 1000;
	box-shadow: 0 4px 12px rgba(0,0,0,0.3);
	display: none;
}

.shortcuts-help.visible {
	display: block;
}

.shortcuts-help h4 {
	margin: 0 0 8px 0;
	color: #4fc3f7;
	font-size: 0.8rem;
}

.shortcuts-help div {
	margin: 4px 0;
}

.shortcuts-help kbd {
	background: #333;
	padding: 2px 6px;
	border-radius: 4px;
	font-family: monospace;
	font-size: 0.7rem;
}

.no-results {
	padding: 20px;
	text-align: center;
	color: #666;
	font-style: italic;
}

.highlight {
	background: rgba(255, 193, 7, 0.3);
	border-radius: 2px;
	padding: 0 2px;
}

.chart-section {
	display: none;
}

#log {
	padding: 16px;
	white-space: pre-wrap;
	word-break: break-all;
	max-height: calc(100vh - 200px);
	overflow-y: auto;
	font-family: inherit;
	scrollbar-width: thin;
	overflow-x: hidden;
}

.entry {
	margin-bottom: 12px;
	padding: 10px 12px;
	border-radius: 8px;
	border-left: 4px solid #555;
	background: linear-gradient(90deg, rgba(40,40,40,0.9) 0%, rgba(30,30,30,0.9) 100%);
	box-shadow: 0 2px 8px rgba(0,0,0,0.2);
	transition: all 0.2s;
	position: relative;
	overflow: hidden;
}

.entry:hover {
	transform: translateX(2px);
	box-shadow: 0 4px 12px rgba(0,0,0,0.3);
}

.entry::before {
	content: '';
	position: absolute;
	top: 0;
	left: 0;
	width: 4px;
	height: 100%;
	background: inherit;
	filter: brightness(1.5);
}

.entry.Catch { 
	border-left-color: #ff5252; 
	background: linear-gradient(90deg, rgba(255,82,82,0.1) 0%, rgba(40,30,30,0.9) 100%);
}
.entry.Out { 
	border-left-color: #ffca28; 
	background: linear-gradient(90deg, rgba(255,202,40,0.1) 0%, rgba(40,35,25,0.9) 100%);
}
.entry.Test { 
	border-left-color: #66bb6a; 
	background: linear-gradient(90deg, rgba(102,187,106,0.1) 0%, rgba(30,40,30,0.9) 100%);
}
.entry.Trace { 
	border-left-color: #42a5f5; 
	background: linear-gradient(90deg, rgba(66,165,245,0.1) 0%, rgba(25,35,45,0.9) 100%);
}
.entry.Walk { 
	border-left-color: #ab47bc; 
	background: linear-gradient(90deg, rgba(171,71,188,0.1) 0%, rgba(35,25,45,0.9) 100%);
}
.entry.CheckStack { 
	border-left-color: #26c6da; 
	background: linear-gradient(90deg, rgba(38,198,218,0.1) 0%, rgba(25,35,40,0.9) 100%);
}
.entry.Mem { 
	border-left-color: #ff7043; 
	background: linear-gradient(90deg, rgba(255,112,67,0.1) 0%, rgba(40,30,25,0.9) 100%);
}
.entry.Tap { 
	border-left-color: #9c27b0; 
	background: linear-gradient(90deg, rgba(156,39,176,0.1) 0%, rgba(35,25,45,0.9) 100%);
}
.entry.Error { 
	border-left-color: #f44336; 
	background: linear-gradient(90deg, rgba(244,67,54,0.1) 0%, rgba(40,25,25,0.9) 100%);
}

.at {
	color: #757575;
	font-size: 0.8em;
	margin-right: 10px;
	font-weight: 500;
}

.tag {
	font-weight: 700;
	margin-right: 8px;
	padding: 2px 6px;
	border-radius: 4px;
	font-size: 0.75rem;
	text-transform: uppercase;
	letter-spacing: 0.5px;
}

.tag.Catch { 
	color: #ff5252; 
	background: rgba(255,82,82,0.2);
}
.tag.Out { 
	color: #ffca28; 
	background: rgba(255,202,40,0.2);
}
.tag.Test { 
	color: #66bb6a; 
	background: rgba(102,187,106,0.2);
}
.tag.Trace { 
	color: #42a5f5; 
	background: rgba(66,165,245,0.2);
}
.tag.Walk { 
	color: #ab47bc; 
	background: rgba(171,71,188,0.2);
}
.tag.CheckStack { 
	color: #26c6da; 
	background: rgba(38,198,218,0.2);
}
.tag.Mem { 
	color: #ff7043; 
	background: rgba(255,112,67,0.2);
}
.tag.Tap { 
	color: #9c27b0; 
	background: rgba(156,39,176,0.2);
}
.tag.Error { 
	color: #f44336; 
	background: rgba(244,67,54,0.2);
}

.body { 
	color: #e0e0e0; 
	line-height: 1.4;
}

.clear-btn {
	position: fixed;
	bottom: 20px;
	right: 20px;
	padding: 10px 16px;
	background: #4fc3f7;
	color: #000;
	border: none;
	border-radius: 20px;
	cursor: pointer;
	font-size: 0.85rem;
	box-shadow: 0 4px 12px rgba(79, 195, 247, 0.3);
	transition: all 0.2s;
	z-index: 1000;
	opacity: 0.6;
}

.clear-btn:hover {
	background: #29b6f6;
	transform: translateY(-2px);
	box-shadow: 0 6px 16px rgba(79, 195, 247, 0.4);
	opacity: 0.85;
}

.entry-count {
	color: #9e9e9e;
	font-size: 0.8rem;
}

</style>
</head>
<body>

<div id="app">
	<div class="header">
		<h1>Miru Dashboard</h1>
		<p>Live debugging logs and traces</p>
	</div>

	<div class="status-bar">
		<div class="status-indicator">
			<div class="status-dot"></div>
			<span>Live</span>
			<span class="entry-count">{{ entryCount }} entries</span>
		</div>
		<div class="count-cards">
			<div class="count-card" 
				 v-for="(count, type) in counts" 
				 :key="type"
				 :class="{ active: activeFilter === type }"
				 @click="setFilter(type)"
				 :data-filter="type">
				<div class="count-dot" :style="{ background: getLogColor(type) }"></div>
				<span class="count-number">{{ count }}</span>
				<span class="count-label">{{ getLogLabel(type) }}</span>
			</div>
		</div>
	</div>

	<div class="chart-section">
		<div class="chart-container">
			<div class="chart-title">Log Distribution Over Time</div>
			<div class="chart-wrapper">
				<canvas ref="lineChart"></canvas>
			</div>
		</div>
	</div>

	<div class="search-bar">
		<input 
			type="text" 
			class="search-input" 
			placeholder="Search logs... (Ctrl+F)"
			v-model="searchQuery"
			@input="handleSearch"
		>
		<div class="controls">
			<button 
				class="control-btn" 
				:class="{ active: autoScroll }"
				@click="toggleAutoScroll"
				title="Auto-scroll (Ctrl+S)"
			>
				Auto-Scroll
			</button>
			<button 
				class="control-btn" 
				@click="exportLogs"
				title="Export logs (Ctrl+E)"
			>
				Export
			</button>
			<button 
				class="control-btn" 
				@click="showShortcuts = !showShortcuts"
				title="Keyboard shortcuts (?)"
			>
				Shortcuts
			</button>
			<button 
				class="control-btn danger"
				@click="clearLogs"
				title="Clear logs (Ctrl+C)"
			>
				Clear
			</button>
		</div>
	</div>

	<div class="shortcuts-help" :class="{ visible: showShortcuts }">
		<h4>Keyboard Shortcuts</h4>
		<div><kbd>Ctrl</kbd> + <kbd>F</kbd> - Focus search</div>
		<div><kbd>Ctrl</kbd> + <kbd>S</kbd> - Toggle auto-scroll</div>
		<div><kbd>Ctrl</kbd> + <kbd>E</kbd> - Export logs</div>
		<div><kbd>Ctrl</kbd> + <kbd>C</kbd> - Clear logs</div>
		<div><kbd>Esc</kbd> - Clear search</div>
		<div><kbd>?</kbd> - Toggle shortcuts</div>
	</div>

	<div id="log">
		<div v-if="filteredEntries.length === 0" class="no-results">
			No logs to display
		</div>
		<div v-for="entry in filteredEntries" 
			 :key="entry.id" 
			 :class="'entry ' + entry.tag"
			 :data-tag="entry.tag"
			 v-html="highlightSearch(formatEntry(entry))">
			<span class="at">{{ entry.at }}</span>
			<span :class="'tag ' + entry.tag">[{{ entry.tag }}]</span>
			<span class="body" v-html="formatBody(entry.body)"></span>
		</div>
	</div>

</div>
<script>

const { createApp } = Vue;

createApp({
	data() {
		return {
			entries: [],
			activeFilter: 'all',
			searchQuery: '',
			autoScroll: true,
			showShortcuts: false,
			counts: {
				all: 0,
				Catch: 0,
				Out: 0,
				Test: 0,
				Trace: 0,
				Walk: 0,
				CheckStack: 0,
				Mem: 0,
				Tap: 0,
				Error: 0
			},
			logColors: {
				all: '#4fc3f7',
				Catch: '#ff5252',
				Out: '#ffca28',
				Test: '#66bb6a',
				Trace: '#42a5f5',
				Walk: '#ab47bc',
				CheckStack: '#26c6da',
				Mem: '#ff7043',
				Tap: '#9c27b0',
				Error: '#f44336'
			},
			logLabels: {
				all: 'All',
				Catch: 'Errors',
				Out: 'Logs',
				Test: 'Tests',
				Trace: 'Traces',
				Walk: 'Walk',
				CheckStack: 'Stack',
				Mem: 'Memory',
				Tap: 'Tap',
				Error: 'IfErr'
			},
			lineChart: null,
			timeData: []
		}
	},
	computed: {
		entryCount() {
			return this.entries.length;
		},
		filteredEntries() {
			let entries = this.entries;
			
			// Filter by type
			if (this.activeFilter !== 'all') {
				entries = entries.filter(entry => entry.tag === this.activeFilter);
			}
			
			// Filter by search query
			if (this.searchQuery.trim()) {
				const query = this.searchQuery.toLowerCase();
				entries = entries.filter(entry => 
					entry.body.toLowerCase().includes(query) ||
					entry.tag.toLowerCase().includes(query) ||
					entry.at.toLowerCase().includes(query)
				);
			}
			
			return entries;
		}
	},
	mounted() {
		this.$nextTick(() => {
			this.connectEventSource();
			this.loadRecentLogs();
			this.setupKeyboardShortcuts();
		});
	},
	methods: {
		getLogColor(type) {
			return this.logColors[type] || '#4fc3f7';
		},
		getLogLabel(type) {
			return this.logLabels[type] || type;
		},
		setFilter(type) {
			this.activeFilter = type;
		},
		formatEntry(entry) {
			return '<span class="at">' + this.escapeHtml(entry.at) + '</span>' +
				'<span class="tag ' + entry.tag + '">[' + entry.tag + ']</span>' +
				'<span class="body">' + this.formatBody(entry.body) + '</span>';
		},
		formatBody(body) {
			return this.escapeHtml(body).replace(/\n/g, '<br>');
		},
		escapeHtml(s) {
			const div = document.createElement('div');
			div.textContent = s;
			return div.innerHTML;
		},
		highlightSearch(html) {
			if (!this.searchQuery.trim()) return html;
			const query = this.escapeHtml(this.searchQuery);
			const regex = new RegExp('(' + query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + ')', 'gi');
			return html.replace(regex, '<span class="highlight">$1</span>');
		},
		handleSearch() {
			// Search is handled reactively by computed property
		},
		toggleAutoScroll() {
			this.autoScroll = !this.autoScroll;
		},
		exportLogs() {
			const logs = this.filteredEntries.map(e => 
				'[' + e.at + '] [' + e.tag + '] ' + e.body
			).join('\n');
			
			const blob = new Blob([logs], { type: 'text/plain' });
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = 'miru-logs-' + new Date().toISOString().slice(0, 19).replace(/:/g, '-') + '.txt';
			a.click();
			URL.revokeObjectURL(url);
		},
		setupKeyboardShortcuts() {
			document.addEventListener('keydown', (e) => {
				// Ctrl/Cmd + F - Focus search
				if ((e.ctrlKey || e.metaKey) && e.key === 'f') {
					e.preventDefault();
					document.querySelector('.search-input').focus();
				}
				
				// Ctrl/Cmd + S - Toggle auto-scroll
				if ((e.ctrlKey || e.metaKey) && e.key === 's') {
					e.preventDefault();
					this.toggleAutoScroll();
				}
				
				// Ctrl/Cmd + E - Export logs
				if ((e.ctrlKey || e.metaKey) && e.key === 'e') {
					e.preventDefault();
					this.exportLogs();
				}
				
				// Ctrl/Cmd + C - Clear logs (with confirmation)
				if ((e.ctrlKey || e.metaKey) && e.key === 'c' && !window.getSelection().toString()) {
					e.preventDefault();
					this.clearLogs();
				}
				
				// Esc - Clear search
				if (e.key === 'Escape') {
					this.searchQuery = '';
					this.showShortcuts = false;
				}
				
				// ? - Toggle shortcuts
				if (e.key === '?' && !e.ctrlKey && !e.metaKey) {
					e.preventDefault();
					this.showShortcuts = !this.showShortcuts;
				}
			});
		},
		addLogEntry(entry) {
			entry.id = Date.now() + Math.random();
			this.entries.push(entry);
			this.counts.all++;
			if (this.counts[entry.tag] !== undefined) {
				this.counts[entry.tag]++;
			}
			if (this.autoScroll) {
				this.scrollToBottom();
			}
		},
		clearLogs() {
			this.entries = [];
			this.counts = {
				all: 0,
				Catch: 0,
				Out: 0,
				Test: 0,
				Trace: 0,
				Walk: 0,
				CheckStack: 0,
				Mem: 0,
				Tap: 0,
				Error: 0
			};
			this.searchQuery = '';
		},
		scrollToBottom() {
			this.$nextTick(() => {
				const logEl = document.getElementById('log');
				logEl.scrollTop = logEl.scrollHeight;
			});
		},
		connectEventSource() {
			const ev = new EventSource('/events');
			ev.onmessage = (e) => {
				const entry = JSON.parse(e.data);
				this.addLogEntry(entry);
			};
		},
		loadRecentLogs() {
			fetch('/api/recent')
				.then(response => response.json())
				.then(entries => {
					entries.forEach(entry => this.addLogEntry(entry));
				})
				.catch(err => console.error('Failed to load recent logs:', err));
		}
	}
}).mount('#app');

</script>

</body>
</html>
`)
