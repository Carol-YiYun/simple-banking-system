// internal/bank/errors.go
//
// 本檔集中定義「領域錯誤（domain errors）」。
// 這些錯誤屬於商業邏輯層級（非系統錯誤），會由上層 HTTP handler 轉換成適當的 HTTP 狀態碼。
// 統一集中管理錯誤類別能確保 API 回傳行為一致、方便測試與維護。

package bank

import "errors"

var (
	// ErrNotFound 代表帳戶不存在。
	// 對應 HTTP 狀態碼 404 Not Found。
	ErrNotFound = errors.New("account not found")

	// ErrBadAmount 代表金額非法（<=0 或初始餘額為負）。
	// 對應 HTTP 狀態碼 400 Bad Request。
	ErrBadAmount = errors.New("amount must be > 0")

	// ErrInsufficient 代表餘額不足，導致提款或轉帳失敗。
	// 對應 HTTP 狀態碼 409 Conflict。
	ErrInsufficient = errors.New("insufficient balance")

	// ErrSameAccount 代表轉帳來源與目標帳戶相同。
	// 對應 HTTP 狀態碼 400 Bad Request。
	ErrSameAccount = errors.New("from and to are same")
)
