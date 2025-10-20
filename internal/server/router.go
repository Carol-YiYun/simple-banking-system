// internal/server/router.go
//
// 本檔負責 HTTP 路由註冊。
// 與 handler.go 分離，讓系統具備更高的擴充彈性：
//   - 可支援 API 版本化（/api/v1, /api/v2）
//   - 可方便插入中介層（middleware，例如驗證、日誌、CORS）
//   - 讓 Server 結構保持單一職責（handler 專注邏輯、router 專注綁定）
//
// 本模組的設計理念：
//   - handler.go 定義「如何處理請求」
//   - router.go 定義「請求如何被導向」
//   - main.go 組裝整體應用（注入 Bank、Storage、Persist Hook）
package server

import "net/http"

// Router 建立並回傳整個 HTTP 處理鏈。
// 採明確路由註冊（非反射式），確保高可讀性與低魔法性。
// 若未來需要版本分支或權限控管，只需在此層新增路由組或 middleware。
func (s *Server) Router() http.Handler {
	v1 := http.NewServeMux()

	// ────────────────
	// API v1 路由定義
	// ────────────────

	// 健康檢查：可供監控或 Docker liveness probe 使用。
	v1.HandleFunc("/health", s.health)

	// 帳戶操作：
	//   - GET  /accounts          → 列出帳戶
	//   - POST /accounts          → 建立帳戶
	v1.HandleFunc("/accounts", s.accounts)

	// 帳戶子操作：
	//   - GET  /accounts/{id}
	//   - POST /accounts/{id}/deposit
	//   - POST /accounts/{id}/withdraw
	//   - GET  /accounts/{id}/logs
	v1.HandleFunc("/accounts/", s.accountSubroutes)

	// 轉帳操作：
	//   - POST /transfer
	v1.HandleFunc("/transfer", s.transfer)

	// ────────────────
	// API Version Mounting
	// ────────────────
	//
	// 將上述所有端點掛在 /api/v1/ 下。
	// 若未來有 /api/v2，只需額外建立一組 mux。
	root := http.NewServeMux()
	root.Handle("/api/v1/", http.StripPrefix("/api/v1", v1))

	// 同時保留根路徑（/），方便本地開發或測試。
	// 若想強制所有 API 都走 /api/v1，可移除此行。
	root.Handle("/", v1)

	return root
}
