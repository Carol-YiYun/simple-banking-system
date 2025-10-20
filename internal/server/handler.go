// internal/server/handler.go
//
// Package server
// ─────────────────────────────────────────────
// 提供 HTTP RESTful 介面，作為 bank 模組的應用層 (Application Layer)。
// 每個 handler 僅負責：
//  1. 接收與驗證 HTTP 請求
//  2. 呼叫 bank 層執行商業邏輯
//  3. 回傳標準化 JSON 回應
//  4. 成功變更狀態後呼叫 s.persist()，將當前銀行狀態寫入 JSON 快照
//
// 此設計使邏輯分層清晰：
//   - bank：純商業邏輯，與 HTTP 無關。
//   - server：處理傳輸層（Transport Layer）。
//   - storage：負責持久化。
//
// 整體遵循「依賴反轉」原則（Bank 不依賴 HTTP，Server 依賴 Bank）。
package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"banking/internal/bank"
)

// Server 為 HTTP 層核心結構：
// - Bank：注入商業邏輯層（銀行核心）。
// - persist：注入持久化鉤子，讓 server 不需關心儲存實作細節（可替換為 DB）。
type Server struct {
	Bank    *bank.Bank
	persist func() error
}

// NewServer 建立新的 HTTP 伺服器。
// persist 可為 nil；若提供則會於每次成功操作後觸發。
func NewServer(b *bank.Bank, persist func() error) *Server {
	return &Server{Bank: b, persist: persist}
}

// accounts 處理：
//   - POST /accounts  → 建立帳戶
//   - GET  /accounts  → 列出所有帳戶
func (s *Server) accounts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			Name    string `json:"name"`
			Balance int64  `json:"balance"`
		}
		// 解析請求內容
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, err, http.StatusBadRequest)
			return
		}
		// 呼叫 Bank 層建立帳戶
		a, err := s.Bank.Create(req.Name, req.Balance)
		if err != nil {
			writeErr(w, err, http.StatusBadRequest)
			return
		}
		// 建立成功 → 回傳 201 Created
		writeJSON(w, http.StatusCreated, a)

		// 持久化快照（非阻塞）
		if s.persist != nil {
			_ = s.persist()
		}

	case http.MethodGet:
		// 列出所有帳戶
		writeJSON(w, http.StatusOK, s.Bank.List())
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// accountSubroutes 處理子路徑：
//
//	GET  /accounts/{id}           → 查詢帳戶
//	POST /accounts/{id}/deposit   → 存款
//	POST /accounts/{id}/withdraw  → 提款
//	GET  /accounts/{id}/logs      → 交易日誌查詢
func (s *Server) accountSubroutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/accounts/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id := parts[0]

	// GET /accounts/{id}
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		a, err := s.Bank.Get(id)
		if err != nil {
			writeErr(w, err, http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, a)
		return
	}

	// 其他子操作
	switch parts[1] {
	case "deposit": // POST /accounts/{id}/deposit
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Amount int64 `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, err, http.StatusBadRequest)
			return
		}
		a, err := s.Bank.Deposit(id, req.Amount)
		if err != nil {
			writeErr(w, err, http.StatusBadRequest)
			return
		}
		// 存款成功後
		writeJSON(w, http.StatusOK, a)
		// 資料持久化
		if s.persist != nil {
			_ = s.persist()
		}

	case "withdraw": // POST /accounts/{id}/withdraw
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Amount int64 `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, err, http.StatusBadRequest)
			return
		}
		a, err := s.Bank.Withdraw(id, req.Amount)
		if err != nil {
			writeErr(w, err, http.StatusBadRequest)
			return
		}
		// 提款成功後
		writeJSON(w, http.StatusOK, a)
		// 資料持久化
		if s.persist != nil {
			_ = s.persist()
		}

	case "logs": // GET /accounts/{id}/logs
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		logs, err := s.Bank.Logs(id)
		if err != nil {
			writeErr(w, err, http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, logs)
	default:
		http.NotFound(w, r)
	}
}

// transfer 處理轉帳：
//
//	POST /transfer  → JSON {From, To, Amount}
//
// 對應題目功能「Able to transfer money from one account to another account」。
// 成功後同時回傳兩帳戶最新餘額。
func (s *Server) transfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		From   string `json:"From"`
		To     string `json:"To"`
		Amount int64  `json:"Amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	// 呼叫 bank 層執行原子轉帳
	if err := s.Bank.Transfer(req.From, req.To, req.Amount); err != nil {
		code := http.StatusBadRequest
		if errors.Is(err, bank.ErrInsufficient) {
			code = http.StatusConflict
		}
		writeErr(w, err, code)
		return
	}

	// 回傳轉帳後的最新帳戶狀態
	fromAcc, _ := s.Bank.Get(req.From)
	toAcc, _ := s.Bank.Get(req.To)

	// 轉帳成功後
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "transfer success",
		"from":    fromAcc,
		"to":      toAcc,
	})
	// 轉帳成功 → 寫入快照
	if s.persist != nil {
		_ = s.persist()
	}
}

// health 提供健康檢查端點：GET /health。
// 可供監控系統或 Docker liveness probe 使用。
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
