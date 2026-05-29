package webui

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sync"

	panoss "pan/internal/oss"
	"pan/internal/osdetect"
)

//go:embed *.html
var assets embed.FS

type event struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Name  string `json:"name"`
	Error string `json:"error,omitempty"`
}

var (
	eventChs   = make(map[chan event]struct{})
	eventMu    sync.Mutex
)

func broadcast(ev event) {
	eventMu.Lock()
	defer eventMu.Unlock()
	for ch := range eventChs {
		select {
		case ch <- ev:
		default:
		}
	}
}

func Start() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	mux := http.NewServeMux()

	htmlFS, _ := fs.Sub(assets, ".")
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, _ := htmlFS.(fs.ReadFileFS).ReadFile("index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	mux.HandleFunc("/api/os", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"os": osdetect.CurrentOS()})
	})

	mux.HandleFunc("/api/download", handleDownload)
	mux.HandleFunc("/api/events", handleEvents)

	url := fmt.Sprintf("http://127.0.0.1:%d", ln.Addr().(*net.TCPAddr).Port)
	openBrowser(url)
	return http.Serve(ln, mux)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start()
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OS  string `json:"os"`
		Dir string `json:"dir"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.OS == "" {
		req.OS = osdetect.CurrentOS()
	}
	if req.Dir == "" {
		req.Dir = "."
	}

	client, err := panoss.NewClient()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	prefix := req.OS + "/"
	keys, err := client.ListAll(prefix)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}

	names := make([]string, len(keys))
	for i, k := range keys {
		names[i] = k[len(prefix):]
	}

	go func() {
		var wg sync.WaitGroup
		for i, key := range keys {
			wg.Add(1)
			go func(idx int, k, name string) {
				defer wg.Done()
				broadcast(event{Type: "start", Index: idx, Name: name})
				if err := client.DownloadFile(k, req.Dir+"/"+name); err != nil {
					broadcast(event{Type: "err", Index: idx, Name: name, Error: err.Error()})
				} else {
					broadcast(event{Type: "ok", Index: idx, Name: name})
				}
			}(i, key, names[i])
		}
		wg.Wait()
	}()

	writeJSON(w, map[string]interface{}{"files": names})
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", 500)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan event, 64)
	eventMu.Lock()
	eventChs[ch] = struct{}{}
	eventMu.Unlock()

	defer func() {
		eventMu.Lock()
		delete(eventChs, ch)
		eventMu.Unlock()
	}()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(ev)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
