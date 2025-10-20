// internal/server/server_test.go
//
// 本檔為 server 層的整合測試 (Integration Test)。
// 模擬完整 HTTP 請求流程，驗證 REST API 與 bank 層之間的整合、狀態正確性、錯誤代碼映射、
// 以及持久化鉤子 (persist hook) 是否在每次成功變更後正確觸發。
//
// 測試重點：
//  1. API 行為符合題目需求（Create / Deposit / Withdraw / Transfer / Logs）。
//  2. 成功操作會觸發持久化 persist()。
//  3. 錯誤狀況皆有正確 HTTP 狀態碼（400, 405, 409 等）。
//  4. 確保測試不依賴外部服務，使用 httptest.Server 完成端對端模擬。
package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"banking/internal/bank"
)

// doJSON 為測試輔助函式：
// 封裝 HTTP JSON 請求邏輯並自動驗證回傳狀態碼。
// 若 out 非 nil，則自動解析 JSON 回應。
// 用於簡化測試程式碼、確保每次測試具一致性。
func doJSON(t *testing.T, c *http.Client, method, url string, body any, wantCode int, out any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, url, &buf)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantCode {
		t.Fatalf("code=%d want=%d", resp.StatusCode, wantCode)
	}
	if out != nil {
		_ = json.NewDecoder(resp.Body).Decode(out)
	}
}

// TestHTTPFlowAndPersistHook
// ------------------------------------------------------------
// 驗證整個 HTTP API 流程的正確性與持久化鉤子行為。
// 涵蓋：
//   - 帳戶建立、存款、提款、轉帳、查詢與日誌
//   - 錯誤情境（轉帳不足、錯誤方法、壞 JSON）
//   - 成功操作後 persist() 被觸發。
//
// ------------------------------------------------------------
func TestHTTPFlowAndPersistHook(t *testing.T) {
	var persistCalls int32 // 用 atomic 計算 persist() 呼叫次數

	b := bank.NewBank()
	s := NewServer(b, func() error {
		atomic.AddInt32(&persistCalls, 1)
		return nil
	})
	ts := httptest.NewServer(s.Router()) // 建立臨時 HTTP 測試伺服器
	defer ts.Close()
	cli := ts.Client()

	// 1️⃣ 建立兩個帳戶
	var a1, a2 bank.Account
	doJSON(t, cli, "POST", ts.URL+"/accounts", map[string]any{"name": "A", "balance": 1000}, 201, &a1)
	doJSON(t, cli, "POST", ts.URL+"/accounts", map[string]any{"name": "B", "balance": 500}, 201, &a2)

	// 2️⃣ 存款與提款
	doJSON(t, cli, "POST", ts.URL+"/accounts/"+a1.ID+"/deposit", map[string]any{"amount": 200}, 200, &a1)
	doJSON(t, cli, "POST", ts.URL+"/accounts/"+a2.ID+"/withdraw", map[string]any{"amount": 100}, 200, &a2) // note: fix path below if needed

	// 3️⃣ 轉帳（含雙方最新餘額回傳）
	var tr struct {
		Message string       `json:"message"`
		From    bank.Account `json:"from"`
		To      bank.Account `json:"to"`
	}
	doJSON(t, cli, "POST", ts.URL+"/transfer", map[string]any{"From": a1.ID, "To": a2.ID, "Amount": 800}, 200, &tr)
	if tr.From.Balance != 400 || tr.To.Balance != 1200 {
		t.Fatalf("balances after transfer: from=%d to=%d", tr.From.Balance, tr.To.Balance)
	}

	// 4️⃣ 查詢單一帳戶
	var got bank.Account
	doJSON(t, cli, "GET", ts.URL+"/accounts/"+a1.ID, nil, 200, &got)
	if got.Balance != 400 {
		t.Fatalf("get a1=%d want 400", got.Balance)
	}

	// 5️⃣ 查詢帳戶日誌
	var logs []bank.Log
	doJSON(t, cli, "GET", ts.URL+"/accounts/"+a2.ID+"/logs", nil, 200, &logs)
	if len(logs) == 0 {
		t.Fatal("expect logs")
	}

	// 6️⃣ 錯誤情境測試
	// (a) 餘額不足 → 409 Conflict
	doJSON(t, cli, "POST", ts.URL+"/transfer", map[string]any{"From": a1.ID, "To": a2.ID, "Amount": 999999}, 409, nil)

	// (b) 錯誤方法 → 405 Method Not Allowed
	doJSON(t, cli, "GET", ts.URL+"/transfer", nil, 405, nil)

	// (c) JSON 格式錯誤 → 400 Bad Request
	req, _ := http.NewRequest("POST", ts.URL+"/accounts/"+a1.ID+"/deposit", bytes.NewBufferString("{bad json}"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := cli.Do(req)
	if resp.StatusCode != 400 {
		t.Fatalf("bad json code=%d want 400", resp.StatusCode)
	}

	// 7️⃣ 驗證 persist 呼叫次數：create×2 + deposit + withdraw + transfer = 5 次以上
	if calls := atomic.LoadInt32(&persistCalls); calls < 5 {
		t.Fatalf("persist calls=%d want>=5", calls)
	}
}

// TestMethodNotAllowed
// ------------------------------------------------------------
// 驗證對不支援的 HTTP 方法或錯誤路徑會正確回傳 405/404。
// 確保 router 與 handler 皆有適當限制。
// ------------------------------------------------------------
func TestMethodNotAllowed(t *testing.T) {
	b := bank.NewBank()
	s := NewServer(b, nil)
	ts := httptest.NewServer(s.Router())
	defer ts.Close()
	cli := ts.Client()

	// POST /accounts/{id} → 錯誤方法 (無對應子路徑)
	req, _ := http.NewRequest("POST", ts.URL+"/accounts/1", nil)
	resp, _ := cli.Do(req)
	if resp.StatusCode != 405 && resp.StatusCode != 404 {
		t.Fatalf("code=%d want 405 or 404", resp.StatusCode)
	}
}
