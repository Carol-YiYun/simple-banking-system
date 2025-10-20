// internal/storage/jsonstore.go
//
// 提供 JSON 快照 (Snapshot) 的序列化與反序列化實作。
// 用於 Bank 系統的輕量持久化方案，可在未接資料庫前保存狀態。
// 採「原子寫入」策略 (atomic write)：先寫入 .tmp 檔，再以 rename() 取代原檔，
// 可避免中途寫入失敗導致檔案損壞，是常見的安全儲存設計模式。
//
// ───────────────────────────────
// 設計理念：
// - **獨立層 (storage)**：不關心商業邏輯，只處理 I/O 序列化。
// - **高穩定性**：使用臨時檔 + rename，確保寫入過程具一致性。
// - **可替換性**：未來若要改成 DB backend，只需替換本層實作。
// ───────────────────────────────
package storage

import (
	"encoding/json"
	"os"
	"time"
)

// LoadSnapshot 讀取指定路徑的 JSON 快照，並解析成 Snapshot 結構。
// 回傳完整快照資料或錯誤。
// 若檔案不存在或格式錯誤，回傳對應錯誤給上層 (通常於系統啟動時呼叫)。
func LoadSnapshot(path string) (Snapshot, error) {
	var snap Snapshot
	f, err := os.Open(path)
	if err != nil {
		return snap, err
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(&snap)
	return snap, err
}

// SaveSnapshot 將 Snapshot 序列化為 JSON 檔案，並採原子方式寫入。
// 流程：
//  1. 設定 Meta.Storage 與當前時間戳。
//  2. 寫入 path+".tmp" 暫存檔。
//  3. 寫入完成後使用 os.Rename() 取代正式檔案。
//
// 這樣設計確保在寫入中斷（例如停電或程式崩潰）時，原檔不會損壞。
func SaveSnapshot(path string, snap Snapshot) error {
	snap.Meta.Storage = "json_snapshot"
	snap.Meta.Timestamp = time.Now()
	tmp := path + ".tmp"

	// 建立暫存檔案
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	// 使用縮排格式輸出，方便人類閱讀（例如除錯或手動檢視）
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(snap); err != nil {
		f.Close()
		return err
	}
	f.Close()

	// 原子替換
	return os.Rename(tmp, path)
}
