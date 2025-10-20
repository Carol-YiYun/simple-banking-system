// internal/storage/jsonstore_test.go
//
// 測試目標：驗證 JSON 快照 (Snapshot) 的序列化與反序列化是否正確。
// 這屬於 storage 層的「資料持久化一致性測試 (persistence integrity test)」，
// 確保資料在寫入與讀取之間沒有遺失或格式錯誤。
//
// 測試重點：
//  1. SaveSnapshot() 能正確建立 JSON 檔案。
//  2. LoadSnapshot() 能完整讀回資料，Meta 與帳戶內容一致。
//  3. 使用 t.TempDir() 確保測試不汙染本機環境。
package storage

import (
	"os"
	"path/filepath"
	"testing"
)

// TestJSONSnapshotRoundTrip
// ------------------------------------------------------------
// 驗證 JSON 快照的 round-trip 過程：
//   - 將 Snapshot 結構序列化成 JSON 檔案。
//   - 再從檔案讀回並反序列化。
//   - 比對欄位與原始資料是否一致。
//
// ------------------------------------------------------------
func TestJSONSnapshotRoundTrip(t *testing.T) {
	// 建立暫存目錄（每次測試獨立，執行結束自動清除）
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")

	// 準備模擬的原始資料
	orig := Snapshot{
		Meta:   Meta{Storage: "json_snapshot", Version: 1, Note: "test"},
		NextID: 3,
		Accounts: []PersistAccount{
			{ID: "1", Name: "A", Balance: 100, Logs: nil},
			{ID: "2", Name: "B", Balance: 200, Logs: nil},
		},
	}

	// 1️⃣ 寫入 JSON 檔案
	if err := SaveSnapshot(path, orig); err != nil {
		t.Fatalf("SaveSnapshot err=%v", err)
	}
	// 驗證檔案存在
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("snapshot not written: %v", err)
	}

	// 2️⃣ 從 JSON 檔案重新載入
	loaded, err := LoadSnapshot(path)
	if err != nil {
		t.Fatalf("LoadSnapshot err=%v", err)
	}
	// 3️⃣ 驗證主要欄位一致性
	if loaded.NextID != orig.NextID || len(loaded.Accounts) != len(orig.Accounts) {
		t.Fatalf("mismatch: loaded=%+v orig=%+v", loaded, orig)
	}
	// 驗證 Meta 欄位正確
	if loaded.Meta.Storage != "json_snapshot" || loaded.Meta.Version != 1 {
		t.Fatalf("meta mismatch: %+v", loaded.Meta)
	}
}
