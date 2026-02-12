package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"ble-printer-bridge/internal/ble"
	"ble-printer-bridge/internal/config"
	"ble-printer-bridge/internal/logging"
	"ble-printer-bridge/internal/printing"
)

type Server struct {
	cfg     *config.Config
	cfgPath string
	log     *logging.Logger
	client  *ble.Client
	cors    *corsConfig
	cfgMu   sync.RWMutex
}

func NewServer(cfg *config.Config, cfgPath string, log *logging.Logger) *Server {
	if err := ble.Enable(); err != nil {
		log.Error("ble enable failed: %v", err)
	} else {
		log.Info("ble adapter enabled")
	}
	srv := &Server{cfg: cfg, cfgPath: cfgPath, log: log, client: &ble.Client{}}
	srv.cors = newCORSConfig(cfg, log)
	return srv
}

func (s *Server) Run() error {
	mux := http.NewServeMux()

	// Health is intentionally unauthenticated
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true})
	})

	// BLE endpoints
	mux.HandleFunc("/ble/scan", s.withRequestLog(s.requireAuth(s.scan)))
	mux.HandleFunc("/ble/connect", s.withRequestLog(s.requireAuth(s.connect)))
	mux.HandleFunc("/ble/disconnect", s.withRequestLog(s.requireAuth(s.disconnect)))
	mux.HandleFunc("/ble/status", s.withRequestLog(s.requireAuth(s.status)))
	mux.HandleFunc("/ble/describe", s.withRequestLog(s.requireAuth(s.describe)))

	// Print endpoints
	mux.HandleFunc("/print/text", s.withRequestLog(s.requireAuth(s.printText)))
	mux.HandleFunc("/print/raw", s.withRequestLog(s.requireAuth(s.printRaw)))

	// Config endpoints
	mux.HandleFunc("/config", s.withRequestLog(s.requireAuth(s.configHandler)))

	cfg := s.configSnapshot()
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	s.log.Info("listening on http://%s", addr)

	return http.ListenAndServe(addr, s.accessLog(corsMiddleware(s.log, s.cors, mux)))
}

func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.log.Info("http %s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}

func (s *Server) withRequestLog(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.log.Info("recv %s %s", r.Method, r.URL.Path)
		next(w, r)
	}
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != s.currentAPIKey() {
			s.log.Warn("unauthorized %s %s", r.Method, r.URL.Path)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("content-type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func (s *Server) scan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := s.configSnapshot()
	var req struct {
		Seconds int `json:"seconds"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Seconds <= 0 {
		req.Seconds = 8
	}

	s.log.Info("ble scan start: seconds=%d filter=%q", req.Seconds, cfg.BLE.DeviceNameContains)
	hits, err := ble.Scan(req.Seconds, cfg.BLE.DeviceNameContains)
	if err != nil {
		s.log.Error("ble scan error: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}
	s.log.Info("ble scan done: found=%d", len(hits))
	writeJSON(w, map[string]any{"ok": true, "found": hits})
}

func (s *Server) connect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Address == "" {
		http.Error(w, `invalid body: {"address":"AA:BB:CC:DD:EE:FF"}`, 400)
		return
	}

	normalizedAddress, err := ble.NormalizeAddress(req.Address)
	if err != nil {
		s.log.Warn("ble connect rejected: raw_address=%q err=%v", req.Address, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.log.Info("ble connect start: raw_address=%q normalized_address=%s", req.Address, normalizedAddress)

	if err := s.client.Connect(normalizedAddress); err != nil {
		s.log.Error("ble connect error: normalized_address=%s err=%v", normalizedAddress, err)
		s.log.Info("ble connect debug scan scheduled: address=%s", normalizedAddress)
		go s.logConnectDebugScan(normalizedAddress)
		http.Error(w, fmt.Sprintf("%v (address=%s; verify the printer is advertising and run /ble/scan)", err, normalizedAddress), 500)
		return
	}
	s.log.Info("ble connect ok: address=%s", normalizedAddress)
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) logConnectDebugScan(address string) {
	const debugScanSeconds = 4
	s.log.Info("ble connect debug scan start: address=%s seconds=%d", address, debugScanSeconds)
	hits, err := ble.Scan(debugScanSeconds, "")
	if err != nil {
		if err == ble.ErrScanBusy {
			s.log.Info("ble connect debug scan skipped: another scan already in progress")
			return
		}
		s.log.Warn("ble connect debug scan failed: %v", err)
		return
	}

	visible := false
	for _, hit := range hits {
		if hit.Address == address {
			visible = true
			break
		}
	}
	s.log.Info("ble connect debug scan done: hits=%d target_visible=%v", len(hits), visible)
	for i, hit := range hits {
		if i >= 8 {
			s.log.Info("ble connect debug scan: additional_hits=%d", len(hits)-i)
			break
		}
		s.log.Info("ble connect debug hit[%d]: address=%s name=%q rssi=%d", i, hit.Address, hit.Name, hit.RSSI)
	}
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	connected := s.client.IsConnected()
	s.log.Info("ble status: connected=%v", connected)
	writeJSON(w, map[string]any{"ok": true, "connected": connected})
}

func (s *Server) disconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.log.Info("ble disconnect start")
	if err := s.client.Disconnect(); err != nil {
		s.log.Error("ble disconnect error: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}
	s.log.Info("ble disconnect ok")
	writeJSON(w, map[string]any{"ok": true, "connected": false})
}

func (s *Server) describe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.log.Info("ble describe start")
	desc, err := s.client.Describe()
	if err != nil {
		s.log.Error("ble describe error: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}
	s.log.Info("ble describe ok: services=%d", len(desc.Services))
	writeJSON(w, map[string]any{"ok": true, "device": desc})
}

func (s *Server) printText(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := s.configSnapshot()
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", 400)
		return
	}

	data := printing.TextReceipt(req.Text)
	s.log.Info("print/text: bytes=%d chunk=%d with_response=%v", len(data), cfg.BLE.ChunkSize, cfg.BLE.WriteWithResponse)

	if err := s.client.Print(
		cfg.BLE.ServiceUUID,
		cfg.BLE.WriteCharacteristicUUID,
		data,
		cfg.BLE.ChunkSize,
		cfg.BLE.WriteWithResponse,
	); err != nil {
		s.log.Error("print/text error: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}
	s.log.Info("print/text ok")
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) printRaw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := s.configSnapshot()
	var req struct {
		Base64 string `json:"base64"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Base64 == "" {
		http.Error(w, "invalid body", 400)
		return
	}
	data, err := base64.StdEncoding.DecodeString(req.Base64)
	if err != nil {
		http.Error(w, "invalid base64", 400)
		return
	}

	s.log.Info("print/raw: bytes=%d chunk=%d with_response=%v", len(data), cfg.BLE.ChunkSize, cfg.BLE.WriteWithResponse)

	if err := s.client.Print(
		cfg.BLE.ServiceUUID,
		cfg.BLE.WriteCharacteristicUUID,
		data,
		cfg.BLE.ChunkSize,
		cfg.BLE.WriteWithResponse,
	); err != nil {
		s.log.Error("print/raw error: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}
	s.log.Info("print/raw ok")
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getConfig(w, r)
	case http.MethodPost:
		s.setConfig(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getConfig(w http.ResponseWriter, r *http.Request) {
	cfg := s.configSnapshot()
	writeJSON(w, map[string]any{"ok": true, "config": cfg})
}

func (s *Server) setConfig(w http.ResponseWriter, r *http.Request) {
	var next config.Config
	if err := json.NewDecoder(r.Body).Decode(&next); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	config.ApplyDefaults(&next)
	if err := config.Save(s.cfgPath, &next); err != nil {
		s.log.Error("config save error: %v", err)
		http.Error(w, "config save failed", http.StatusInternalServerError)
		return
	}
	s.replaceConfig(&next)
	s.log.Info("config updated")
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) currentAPIKey() string {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return s.cfg.Auth.ApiKey
}

func (s *Server) configSnapshot() config.Config {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return *s.cfg
}

func (s *Server) replaceConfig(cfg *config.Config) {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	s.cfg = cfg
	s.cors = newCORSConfig(cfg, s.log)
}
