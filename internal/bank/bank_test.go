// internal/bank/bank_test.go
//
// 本檔為 Bank 模組的單元與整合測試。
// 覆蓋題目要求的所有功能：帳戶建立、存提款、轉帳、餘額驗證、交易日誌、原子性與快照。
// 所有測試皆為 in-memory 執行，不依賴外部服務或資料庫。

package bank

import (
	"errors"
	"sync"
	"testing"
)

// get 為小工具：安全取出帳戶狀態。
// 若發生錯誤，立即讓測試失敗（方便多測例共用）。
func get(t *testing.T, b *Bank, id string) *Account {
	t.Helper()
	a, err := b.Get(id)
	if err != nil {
		t.Fatalf("Get(%s) err=%v", id, err)
	}
	return a
}

// TestCreateAndListGet 驗證帳戶建立、查詢與列出功能。
// 涵蓋：唯一 ID、名稱與初始餘額正確性。
func TestCreateAndListGet(t *testing.T) {
	b := NewBank()
	a1, err := b.Create("A", 1000)
	if err != nil {
		t.Fatal(err)
	}
	a2, err := b.Create("B", 500)
	if err != nil {
		t.Fatal(err)
	}
	// ID 必須唯一且非空
	if a1.ID == a2.ID || a1.ID == "" || a2.ID == "" {
		t.Fatalf("ids should be unique and non-empty: %q %q", a1.ID, a2.ID)
	}
	// List() 應回傳兩筆帳戶
	all := b.List()
	if len(all) != 2 {
		t.Fatalf("List len=%d want=2", len(all))
	}
	// 驗證個別帳戶資訊正確
	g1 := get(t, b, a1.ID)
	if g1.Name != "A" || g1.Balance != 1000 {
		t.Fatalf("got=%+v want name=A balance=1000", g1)
	}
}

// TestCreateNegativeBalance 驗證建立帳戶時不得為負餘額。
// 對應題目：「Account balance cannot be negative」
func TestCreateNegativeBalance(t *testing.T) {
	b := NewBank()
	if _, err := b.Create("A", -1); !errors.Is(err, ErrBadAmount) {
		t.Fatalf("want ErrBadAmount, got %v", err)
	}
}

// TestDepositWithdraw 測試存款與提款功能。
// 涵蓋正常路徑與錯誤條件（非法金額、餘額不足）。
func TestDepositWithdraw(t *testing.T) {
	b := NewBank()
	a, _ := b.Create("A", 100)

	// ✅ 正常存提款
	if _, err := b.Deposit(a.ID, 50); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Withdraw(a.ID, 30); err != nil {
		t.Fatal(err)
	}
	if bal := get(t, b, a.ID).Balance; bal != 120 {
		t.Fatalf("balance=%d want=120", bal)
	}

	// ❌ 錯誤金額：0 或負數
	if _, err := b.Deposit(a.ID, 0); !errors.Is(err, ErrBadAmount) {
		t.Fatalf("expect ErrBadAmount, got %v", err)
	}
	if _, err := b.Withdraw(a.ID, -1); !errors.Is(err, ErrBadAmount) {
		t.Fatalf("expect ErrBadAmount, got %v", err)
	}

	// ❌ 餘額不足
	if _, err := b.Withdraw(a.ID, 9999); !errors.Is(err, ErrInsufficient) {
		t.Fatalf("expect ErrInsufficient, got %v", err)
	}
}

// TestTransfer 驗證轉帳邏輯。
// 涵蓋：正常轉帳、相同帳戶、餘額不足三種情境。
func TestTransfer(t *testing.T) {
	b := NewBank()
	a1, _ := b.Create("A", 1000)
	a2, _ := b.Create("B", 500)

	// ✅ 正常轉帳
	if err := b.Transfer(a1.ID, a2.ID, 300); err != nil {
		t.Fatal(err)
	}
	if got := get(t, b, a1.ID).Balance; got != 700 {
		t.Fatalf("a1=%d want=700", got)
	}
	if got := get(t, b, a2.ID).Balance; got != 800 {
		t.Fatalf("a2=%d want=800", got)
	}

	// ❌ 相同帳戶不得轉帳
	if err := b.Transfer(a1.ID, a1.ID, 1); !errors.Is(err, ErrSameAccount) {
		t.Fatalf("expect ErrSameAccount, got %v", err)
	}

	// ❌ 餘額不足
	if err := b.Transfer(a1.ID, a2.ID, 99999); !errors.Is(err, ErrInsufficient) {
		t.Fatalf("expect ErrInsufficient, got %v", err)
	}
}

// TestTransferBadAmount 驗證轉帳金額必須大於 0。
func TestTransferBadAmount(t *testing.T) {
	b := NewBank()
	a1, _ := b.Create("A", 100)
	a2, _ := b.Create("B", 100)

	for _, amt := range []int64{0, -5} {
		if err := b.Transfer(a1.ID, a2.ID, amt); !errors.Is(err, ErrBadAmount) {
			t.Fatalf("amt=%d want ErrBadAmount, got %v", amt, err)
		}
	}
}

// TestConcurrentTransfersAtomicity 驗證高併發下轉帳原子性。
// 對應題目：「Support atomic transaction」。
// 模擬雙方帳戶各 200 次交互轉帳後，總額應不變且皆非負。
func TestConcurrentTransfersAtomicity(t *testing.T) {
	b := NewBank()
	a1, _ := b.Create("A", 1000)
	a2, _ := b.Create("B", 1000)

	const n = 200
	var wg sync.WaitGroup
	wg.Add(2 * n)

	// 並行模擬 A→B
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if err := b.Transfer(a1.ID, a2.ID, 1); err != nil {
				t.Errorf("A->B: %v", err)
			}
		}()
	}
	// 並行模擬 B→A
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if err := b.Transfer(a2.ID, a1.ID, 1); err != nil {
				t.Errorf("B->A: %v", err)
			}
		}()
	}
	wg.Wait()

	ga1, _ := b.Get(a1.ID)
	ga2, _ := b.Get(a2.ID)

	// 確認無負餘額
	if ga1.Balance < 0 || ga2.Balance < 0 {
		t.Fatalf("negative balance: a1=%d a2=%d", ga1.Balance, ga2.Balance)
	}
	// 總資金恆等於初始 2000
	if total := ga1.Balance + ga2.Balance; total != 2000 {
		t.Fatalf("total=%d want 2000", total)
	}
}

// TestLogs 驗證每筆操作都會生成正確的交易日誌。
// 對應題目：「Generate transaction logs for each account transfer」
func TestLogs(t *testing.T) {
	b := NewBank()
	a1, _ := b.Create("A", 1000)
	a2, _ := b.Create("B", 0)

	// 模擬存、提、轉帳
	_, _ = b.Deposit(a2.ID, 200)
	_, _ = b.Withdraw(a2.ID, 50)
	_ = b.Transfer(a1.ID, a2.ID, 300)

	logs1, err := b.Logs(a1.ID)
	if err != nil {
		t.Fatal(err)
	}
	logs2, err := b.Logs(a2.ID)
	if err != nil {
		t.Fatal(err)
	}

	// A 僅有一筆轉出紀錄
	if len(logs1) != 1 || logs1[0].Direction != "out" || logs1[0].Amount != 300 || logs1[0].CounterID != a2.ID {
		t.Fatalf("logs1 unexpected: %+v", logs1)
	}
	// B 應有三筆紀錄：存入、提領、轉入
	if len(logs2) != 3 {
		t.Fatalf("logs2 len=%d want=3", len(logs2))
	}
	// 只檢查方向與金額，不比對時間
	if logs2[0].Direction != "in" || logs2[0].Amount != 200 {
		t.Fatalf("logs2[0] unexpected: %+v", logs2[0])
	}
	if logs2[1].Direction != "out" || logs2[1].Amount != 50 {
		t.Fatalf("logs2[1] unexpected: %+v", logs2[1])
	}
	if logs2[2].Direction != "in" || logs2[2].Amount != 300 || logs2[2].CounterID != a1.ID {
		t.Fatalf("logs2[2] unexpected: %+v", logs2[2])
	}

	// 驗證時間欄位皆有設置
	if logs1[0].Time.IsZero() || logs2[2].Time.IsZero() {
		t.Fatalf("time field should be set for logs")
	}

}

// TestConcurrentDepositsRaceSafety 驗證多執行緒同時存款仍具資料一致性。
// 對應題目：「Support atomic transaction」
func TestConcurrentDepositsRaceSafety(t *testing.T) {
	b := NewBank()
	a, _ := b.Create("A", 0)

	const workers = 100
	const amt = int64(1)

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			if _, err := b.Deposit(a.ID, amt); err != nil {
				t.Errorf("deposit err: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := get(t, b, a.ID).Balance; got != workers*amt {
		t.Fatalf("balance=%d want=%d", got, workers*amt)
	}
}

// TestSnapshotRestore 驗證快照儲存與還原功能。
// 對應題目：「data persistence (in-memory or snapshot)」。
// 確保餘額與交易日誌在還原後完全一致。
func TestSnapshotRestore(t *testing.T) {
	b := NewBank()
	a1, _ := b.Create("A", 1000)
	a2, _ := b.Create("B", 500)
	_, _ = b.Deposit(a1.ID, 200)
	_, _ = b.Withdraw(a2.ID, 100)
	_ = b.Transfer(a1.ID, a2.ID, 800)

	snap := b.Snapshot()

	// 新的 Bank 從快照復原
	b2 := NewBank()
	b2.Restore(snap)

	// 驗證餘額一致
	if get(t, b2, a1.ID).Balance != 400 {
		t.Fatalf("restored a1 balance want=400 got=%d", get(t, b2, a1.ID).Balance)
	}
	if get(t, b2, a2.ID).Balance != 1200 {
		t.Fatalf("restored a2 balance want=1200 got=%d", get(t, b2, a2.ID).Balance)
	}

	// 日誌數量也應一致
	l1, _ := b.Logs(a1.ID)
	l1r, _ := b2.Logs(a1.ID)
	if len(l1) != len(l1r) {
		t.Fatalf("logs count mismatch a1: %d vs %d", len(l1), len(l1r))
	}
	l2, _ := b.Logs(a2.ID)
	l2r, _ := b2.Logs(a2.ID)
	if len(l2) != len(l2r) {
		t.Fatalf("logs count mismatch a2: %d vs %d", len(l2), len(l2r))
	}
}
