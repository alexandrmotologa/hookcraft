package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexandrmotologa/hookcraft/internal/config"
	"github.com/alexandrmotologa/hookcraft/internal/git"
	"github.com/alexandrmotologa/hookcraft/internal/runner"
	"github.com/alexandrmotologa/hookcraft/internal/secrets"
)

// Server provides an embedded HTTP API and dashboard.
type Server struct {
	RepoRoot   string
	GitRunner  *git.Runner
	Config     *config.Config
	ConfigPath string
	Assets     embed.FS
	Port       int
}

// StatusResponse represents general repository and hook health.
type StatusResponse struct {
	RepoRoot    string          `json:"repo_root"`
	Branch      string          `json:"branch"`
	HookStatus  map[string]bool `json:"hook_status"`
	StagedFiles []string        `json:"staged_files"`
	StagedCount int             `json:"staged_count"`
	HasUnstaged bool            `json:"has_unstaged"`
	ConfigPath  string          `json:"config_path"`
	SecretCount int             `json:"secret_rule_count"`
	Version     string          `json:"version"`
}

// NewServer creates a new Server instance.
func NewServer(repoRoot string, gitRunner *git.Runner, cfg *config.Config, configPath string, assets embed.FS, port int) *Server {
	if port <= 0 {
		port = 4173
	}
	return &Server{
		RepoRoot:   repoRoot,
		GitRunner:  gitRunner,
		Config:     cfg,
		ConfigPath: configPath,
		Assets:     assets,
		Port:       port,
	}
}

// Start launches the HTTP server.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/diff", s.handleDiff)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/scan", s.handleScan)

	// Static web assets handler (with SPA fallback)
	s.setupStaticAssets(mux)

	addr := fmt.Sprintf(":%d", s.Port)
	fmt.Printf("\n[hookcraft studio] Listening at http://localhost:%d\n", s.Port)
	fmt.Printf("[hookcraft studio] Press Ctrl+C to stop server.\n\n")

	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	branch, _ := s.GitRunner.Exec("rev-parse", "--abbrev-ref", "HEAD")
	branch = strings.TrimSpace(branch)

	hooks, _ := s.GitRunner.GetHookStatus()
	staged, _ := s.GitRunner.GetStagedFiles()
	hasUnstaged, _ := s.GitRunner.HasUnstagedChanges()

	resp := StatusResponse{
		RepoRoot:    s.RepoRoot,
		Branch:      branch,
		HookStatus:  hooks,
		StagedFiles: staged,
		StagedCount: len(staged),
		HasUnstaged: hasUnstaged,
		ConfigPath:  s.ConfigPath,
		SecretCount: len(secrets.SecretRules),
		Version:     "1.0.0",
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	history, err := runner.LoadHistory(s.RepoRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, history)
}

func (s *Server) handleDiff(w http.ResponseWriter, r *http.Request) {
	rawDiff, err := s.GitRunner.GetStagedDiff()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	hunks, err := s.GitRunner.GetParsedStagedHunks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := map[string]interface{}{
		"raw_diff": rawDiff,
		"hunks":    hunks,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	var rawYAML string
	if s.ConfigPath != "" {
		if data, err := os.ReadFile(s.ConfigPath); err == nil {
			rawYAML = string(data)
		}
	} else {
		rawYAML = config.GenerateStarterYAML(s.RepoRoot)
	}

	resp := map[string]interface{}{
		"config":   s.Config,
		"raw_yaml": rawYAML,
		"path":     s.ConfigPath,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	hunks, err := s.GitRunner.GetParsedStagedHunks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	scanner := secrets.NewScanner(s.Config.Settings.EntropyThreshold)
	findings := scanner.ScanHunks(hunks)

	resp := map[string]interface{}{
		"findings":    findings,
		"count":       len(findings),
		"threshold":   s.Config.Settings.EntropyThreshold,
		"rules_count": len(scanner.Rules),
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) setupStaticAssets(mux *http.ServeMux) {
	// Try embedded dist first
	sub, err := fs.Sub(s.Assets, "dist")
	if err == nil {
		fileServer := http.FileServer(http.FS(sub))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Don't intercept API routes
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}

			// Check if file exists in embedded fs
			f, err := sub.Open(strings.TrimPrefix(r.URL.Path, "/"))
			if err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}

			// SPA fallback: serve index.html
			indexFile, err := sub.Open("index.html")
			if err == nil {
				_ = indexFile.Close()
				r.URL.Path = "/"
				fileServer.ServeHTTP(w, r)
				return
			}

			// If no index.html found in embedded fs, serve standalone fallback HTML
			s.serveFallbackHTML(w)
		})
		return
	}

	// Fallback to local web/dist or standalone HTML
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		localDist := filepath.Join(s.RepoRoot, "web", "dist")
		if _, err := os.Stat(filepath.Join(localDist, "index.html")); err == nil {
			http.FileServer(http.Dir(localDist)).ServeHTTP(w, r)
			return
		}

		s.serveFallbackHTML(w)
	})
}

func (s *Server) serveFallbackHTML(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>HookCraft Studio</title>
  <style>
    body { font-family: ui-sans-serif, system-ui, -apple-system, sans-serif; background: #0f172a; color: #f8fafc; padding: 2rem; }
    h1 { color: #38bdf8; }
    .card { background: #1e293b; border-radius: 8px; padding: 1.5rem; margin-top: 1rem; border: 1px solid #334155; }
    pre { background: #020617; padding: 1rem; border-radius: 6px; overflow-x: auto; color: #a5f3fc; }
  </style>
</head>
<body>
  <h1>HookCraft Studio (Development Mode)</h1>
  <p>Ultra-Fast Zero-Dependency Git Hook Orchestrator and Secret Leak Shield</p>
  <div class="card">
    <h3>Available REST API Endpoints:</h3>
    <ul>
      <li><a href="/api/status" style="color: #38bdf8">/api/status</a> — Repository health and hook installation status</li>
      <li><a href="/api/history" style="color: #38bdf8">/api/history</a> — Historical benchmark runs and execution timings</li>
      <li><a href="/api/diff" style="color: #38bdf8">/api/diff</a> — Staged diff hunks</li>
      <li><a href="/api/config" style="color: #38bdf8">/api/config</a> — Active HookCraft configuration</li>
      <li><a href="/api/scan" style="color: #38bdf8">/api/scan</a> — Real-time secret leak audit</li>
    </ul>
  </div>
</body>
</html>`
	_, _ = w.Write([]byte(html))
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
