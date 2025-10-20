// internal/server/response.go
//
// 本檔負責統一 HTTP 回應格式。
// 透過集中管理 JSON 與錯誤輸出，可確保整個 REST API 的一致性與可維護性。
// 設計理念：
//   - 「成功回應」使用標準 JSON 編碼（Content-Type: application/json）。
//   - 「錯誤回應」統一由 writeErr 輸出，方便日後集中擴充（例如自訂錯誤結構）。
//
// 若未來要支援更完整的 API 響應規範（如 RFC 7807 problem+json），
// 僅需在此檔擴充，無須修改各個 handler。
package server

import (
	"encoding/json"
	"net/http"
)

// writeJSON 統一輸出成功回應。
// - code：HTTP 狀態碼（例如 200, 201）
// - v：可被 JSON 序列化的物件（map、struct、slice 皆可）
// 實務上所有成功路徑皆應透過此函式回傳，以維持一致格式。
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErr 統一輸出錯誤回應。
// - err.Error()：直接輸出為純文字訊息
// - code：HTTP 狀態碼（400、404、409 等）
//
// 此為簡化版；若要回傳 JSON 格式錯誤（例如 {"error": "..."}），
// 可擴充為：
//
//	writeJSON(w, code, map[string]string{"error": err.Error()})
func writeErr(w http.ResponseWriter, err error, code int) {
	http.Error(w, err.Error(), code)
}
