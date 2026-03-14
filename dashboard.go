package miru

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const dashboardMaxEntries = 500

// LogEntry is one item sent to the dashboard (and stored for recent).
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
	h.mu.Lock()
	h.entries = append(h.entries, e)
	if len(h.entries) > dashboardMaxEntries {
		h.entries = h.entries[len(h.entries)-dashboardMaxEntries:]
	}
	data, _ := json.Marshal(e)
	for ch := range h.clients {
		select {
		case ch <- data:
		default:
			// client slow, skip this event
		}
	}
	h.mu.Unlock()
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
	addr := ":" + fmt.Sprint(port)
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
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() { _ = srv.ListenAndServe() }()
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
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}
	recent, ch := h.Subscribe()
	defer h.Unsubscribe(ch)
	for _, e := range recent {
		data, _ := json.Marshal(e)
		_, _ = w.Write([]byte("data: "))
		_, _ = w.Write(data)
		_, _ = w.Write([]byte("\n\n"))
		flusher.Flush()
	}
	for data := range ch {
		_, _ = w.Write([]byte("data: "))
		_, _ = w.Write(data)
		_, _ = w.Write([]byte("\n\n"))
		flusher.Flush()
	}
}

var dashboardHTML = []byte(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Miru Dashboard</title>
  <style>
    * { box-sizing: border-box; }
    body { font-family: ui-monospace, monospace; font-size: 13px; margin: 0; background: #1a1b1e; color: #c5c8c6; min-height: 100vh; }
    .header { padding: 10px 14px; border-bottom: 1px solid #3e4451; background: #282a2e; }
    .header h1 { margin: 0; font-size: 1.1rem; font-weight: 600; color: #b5bd68; }
    .header p { margin: 4px 0 0; font-size: 0.85rem; color: #707880; }
    #log { padding: 12px; white-space: pre-wrap; word-break: break-all; max-height: calc(100vh - 80px); overflow-y: auto; }
    .entry { margin-bottom: 8px; padding: 6px 8px; border-radius: 4px; border-left: 3px solid #5c6370; background: #21252b; }
    .entry.Catch { border-left-color: #e06c75; }
    .entry.Out { border-left-color: #e5c07b; }
    .entry.Test { border-left-color: #98c379; }
    .entry.Trace { border-left-color: #61afef; }
    .entry.Walk { border-left-color: #c678dd; }
    .entry.CheckStack { border-left-color: #56b6c2; }
    .at { color: #5c6370; font-size: 0.9em; margin-right: 8px; }
    .tag { font-weight: 600; margin-right: 6px; }
    .tag.Catch { color: #e06c75; }
    .tag.Out { color: #e5c07b; }
    .tag.Test { color: #98c379; }
    .tag.Trace { color: #61afef; }
    .tag.Walk { color: #c678dd; }
    .tag.CheckStack { color: #56b6c2; }
    .body { color: #abb2bf; }
  </style>
</head>
<body>
  <div class="header">
    <h1>Miru Dashboard</h1>
    <p>Live logs and traces</p>
  </div>
  <div id="log"></div>
  <script>
    const logEl = document.getElementById('log');
    const ev = new EventSource('/events');
    ev.onmessage = function(e) {
      const entry = JSON.parse(e.data);
      const div = document.createElement('div');
      div.className = 'entry ' + entry.tag;
      div.innerHTML = '<span class="at">' + escapeHtml(entry.at) + '</span><span class="tag ' + entry.tag + '">[' + entry.tag + ']</span><span class="body">' + escapeHtml(entry.body).replace(/\n/g, '<br>') + '</span>';
      logEl.appendChild(div);
      logEl.scrollTop = logEl.scrollHeight;
    };
    function escapeHtml(s) {
      const d = document.createElement('div');
      d.textContent = s;
      return d.innerHTML;
    }
  </script>
</body>
</html>
`)
