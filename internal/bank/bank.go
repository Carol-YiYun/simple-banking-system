// internal/bank/bank.go

// Package bank 定義核心商業邏輯：帳戶建立、存款、提款、轉帳、查詢與交易日誌。
// 採用單一互斥鎖 (sync.Mutex) 保障所有狀態變更「原子且序列化」，避免競爭條件。
// 金額以 int64 的最小貨幣單位（如分）儲存，避免浮點誤差。
package bank

import (
	"banking/internal/storage"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Bank 為聚合根 (Aggregate Root)：管理全系統帳戶。
// - mu：序列化所有讀寫，確保跨帳戶操作（轉帳）原子完成。
// - nextID：以原子遞增產生帳戶 ID，避免並發碰撞。
// - accts：帳戶索引表（ID → *Account），內部所有指標只在臨界區內修改。
type Bank struct {
	mu     sync.Mutex
	nextID int64
	accts  map[string]*Account
}

// NewBank 建立空白銀行實例（僅就緒的 in-memory 狀態，無外部依賴）。
func NewBank() *Bank {
	return &Bank{accts: make(map[string]*Account)}
}

// newID 回傳唯一遞增字串 ID。
// 使用 atomic 避免在高併發下 ID 碰撞；真正寫入 map 仍在 mu 保護下。
func (b *Bank) newID() string {
	id := atomic.AddInt64(&b.nextID, 1)
	return fmt.Sprintf("%d", id)
}

// Create 以名稱與初始餘額建立帳戶；初始餘額不得為負。
// 回傳淺拷貝（非內部指標）避免呼叫端越權修改內部狀態。
func (b *Bank) Create(name string, balance int64) (*Account, error) {
	if balance < 0 {
		return nil, ErrBadAmount
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	id := b.newID()
	a := &Account{ID: id, Name: name, Balance: balance}
	b.accts[id] = a
	return a, nil
}

// Get 依 ID 取得帳戶的目前快照；若不存在回傳 ErrNotFound。
// 回傳的是值拷貝，避免外部直接改寫內部指標。
func (b *Bank) Get(id string) (*Account, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	a, ok := b.accts[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *a
	return &cp, nil
}

// List 回傳所有帳戶的淺拷貝快照；不暴露內部指標，維持封裝。
func (b *Bank) List() []*Account {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]*Account, 0, len(b.accts))
	for _, a := range b.accts {
		cp := *a
		out = append(out, &cp)
	}
	return out
}

// Deposit 存款：金額需 > 0；若帳戶不存在回傳 ErrNotFound。
// 於臨界區內同時更新餘額與追加日誌，確保兩者一致性。
func (b *Bank) Deposit(id string, amt int64) (*Account, error) {
	if amt <= 0 {
		return nil, ErrBadAmount
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	a, ok := b.accts[id]
	if !ok {
		return nil, ErrNotFound
	}
	a.Balance += amt
	a.Logs = append(a.Logs, Log{Time: time.Now(), Amount: amt, Direction: "in", Note: "deposit"})
	cp := *a
	return &cp, nil
}

// Withdraw 提款：金額需 > 0 且不得超過餘額（維持非負）；不存在則 ErrNotFound。
// 同樣於臨界區內一併更新餘額與日誌，避免部分成功。
func (b *Bank) Withdraw(id string, amt int64) (*Account, error) {
	if amt <= 0 {
		return nil, ErrBadAmount
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	a, ok := b.accts[id]
	if !ok {
		return nil, ErrNotFound
	}
	if a.Balance < amt {
		return nil, ErrInsufficient
	}
	a.Balance -= amt
	a.Logs = append(a.Logs, Log{Time: time.Now(), Amount: amt, Direction: "out", Note: "withdraw"})
	cp := *a
	return &cp, nil
}

// Transfer 轉帳為「單一臨界區內」的原子操作：
// 1) 檢核參數與帳戶存在性 → 2) 檢查餘額 → 3) 同步扣款與入帳 → 4) 同步雙邊日誌。
// 任一步驟失敗皆不會改變任何帳戶狀態。
func (b *Bank) Transfer(fromID, toID string, amt int64) error {
	if amt <= 0 {
		return ErrBadAmount
	}
	if fromID == toID {
		return ErrSameAccount
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	from, ok1 := b.accts[fromID]
	to, ok2 := b.accts[toID]
	if !ok1 || !ok2 {
		return ErrNotFound
	}
	if from.Balance < amt {
		return ErrInsufficient
	}

	from.Balance -= amt
	to.Balance += amt

	now := time.Now()
	from.Logs = append(from.Logs, Log{Time: now, Amount: amt, Direction: "out", CounterID: toID, Note: "transfer"})
	to.Logs = append(to.Logs, Log{Time: now, Amount: amt, Direction: "in", CounterID: fromID, Note: "transfer"})
	return nil
}

// Logs 回傳指定帳戶的交易日誌（值拷貝），避免外部修改內部切片。
func (b *Bank) Logs(id string) ([]Log, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	a, ok := b.accts[id]
	if !ok {
		return nil, ErrNotFound
	}
	out := make([]Log, len(a.Logs))
	copy(out, a.Logs)
	return out, nil
}

// Snapshot 匯出銀行狀態到可持久化的 storage.Snapshot：
// - 包含 nextID 與所有帳戶（含日誌）
// - _meta.section 內寫入 storage 類型與版本，便於未來 schema 遷移/換後端存儲。
func (b *Bank) Snapshot() storage.Snapshot {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := storage.Snapshot{
		Meta: storage.Meta{
			Storage: "json_snapshot",
			Version: 1,
			Note:    "Can be replaced by database backend in the future.",
		},
		NextID: b.nextID,
	}
	for _, a := range b.accts {
		s.Accounts = append(s.Accounts, storage.PersistAccount{
			ID: a.ID, Name: a.Name, Balance: a.Balance, Logs: toAnySlice(a.Logs),
		})
	}
	return s
}

// Restore 由 storage.Snapshot 還原銀行狀態：重建 nextID 與帳戶 map。
// 為確保未來向後相容，對未知欄位採用 JSON 中介轉換（logs）。
func (b *Bank) Restore(s storage.Snapshot) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID = s.NextID
	b.accts = make(map[string]*Account)
	for _, pa := range s.Accounts {
		a := &Account{ID: pa.ID, Name: pa.Name, Balance: pa.Balance}
		for _, l := range pa.Logs {
			var log Log
			j, _ := json.Marshal(l)
			_ = json.Unmarshal(j, &log)
			a.Logs = append(a.Logs, log)
		}
		b.accts[a.ID] = a
	}
}

// toAnySlice 將型別化切片轉為 []any，供快照序列化使用。
// 不做深拷貝（元素為值類型），符合 JSON 編碼需求。
func toAnySlice[T any](in []T) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}
