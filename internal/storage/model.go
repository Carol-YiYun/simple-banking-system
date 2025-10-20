// internal/storage/model.go
//
// 定義「資料持久化層 (storage layer)」的結構模型。
// 該層的責任是提供 Bank 系統的資料序列化格式（目前為 JSON），
// 並保存必要的中繼資訊 (Meta)，以便版本控制與未來可擴充為資料庫後端。
//
// ───────────────────────────────
// 設計理念：
// - **關注分離**：此層僅定義資料結構，不涉入商業邏輯。
// - **可演進性**：Meta 保留版本與時間戳，可支援多種儲存實作（JSON / DB）。
// - **可追溯性**：所有快照具備建立時間與註解，便於除錯或審計。
// ───────────────────────────────
package storage

import "time"

// Meta 為所有持久化快照的中繼資料 (metadata)。
// 用於記錄儲存方式、版本、建立時間與說明。
// 可協助後續進行格式升級、除錯或追蹤快照來源。
type Meta struct {
	Storage   string    `json:"storage"`        // 儲存類型，例如 "json_snapshot"
	Version   int       `json:"version"`        // 結構版本號，用於未來升級時比對
	Timestamp time.Time `json:"timestamp"`      // 快照建立時間
	Note      string    `json:"note,omitempty"` // 備註欄，可選，用於人工說明
}

// PersistAccount 為帳戶在儲存層的序列化格式。
// 不含同步鎖或方法，僅保存資料狀態，確保可安全序列化至 JSON 或資料庫。
type PersistAccount struct {
	ID      string `json:"id"`      // 帳戶唯一 ID
	Name    string `json:"name"`    // 帳戶名稱
	Balance int64  `json:"balance"` // 帳戶餘額，以最小貨幣單位儲存
	Logs    []any  `json:"logs"`    // 交易日誌，以任意型別儲存（JSON 可直接還原）
}

// Snapshot 為 Bank 狀態的完整快照。
// 包含所有帳戶資料與中繼資訊，用於整體載入與保存。
// 每次程式結束或狀態改變時可重新產出，確保系統一致性。
type Snapshot struct {
	Meta     Meta             `json:"_meta"`    // 中繼資料（儲存資訊與版本）
	NextID   int64            `json:"next_id"`  // 下一個帳戶可用 ID
	Accounts []PersistAccount `json:"accounts"` // 帳戶清單（序列化後的純資料）
}
