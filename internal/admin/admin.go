package admin

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/tony19053000/wallet-api/internal/httpx"
	"github.com/tony19053000/wallet-api/internal/wallet"
)

type AdminWallet struct {
	ID                  string    `json:"id"`
	UserID              string    `json:"user_id"`
	UserName            string    `json:"user_name"`
	UserEmail           string    `json:"user_email"`
	UserHandle          string    `json:"user_handle"`
	KYCLevel            string    `json:"kyc_level"`
	BalancePaise        int64     `json:"balance_paise"`
	Status              string    `json:"status"`
	DailySendLimitPaise int64     `json:"daily_send_limit_paise"`
	CreatedAt           time.Time `json:"created_at"`
}

type AdminWalletDetail struct {
	Wallet       AdminWallet          `json:"wallet"`
	Transactions []wallet.Transaction `json:"transactions"`
}

type AdminMetrics struct {
	TotalBalancesHeldPaise int64        `json:"total_balances_held_paise"`
	TodayVolumePaise       int64        `json:"today_volume_paise"`
	TotalTransfersCount    int64        `json:"total_transfers_count"`
	FailedTransferRate     float64      `json:"failed_transfer_rate"`
	TopSenders             []TopSender  `json:"top_senders"`
}

type TopSender struct {
	UserID      string `json:"user_id"`
	UserName    string `json:"user_name"`
	UserHandle  string `json:"user_handle"`
	TotalPaise  int64  `json:"total_paise"`
	Count       int64  `json:"count"`
}

type LedgerCheckResponse struct {
	SumOfBalances int64 `json:"sum_of_balances"`
	SumOfLedger   int64 `json:"sum_of_ledger"`
	Matches       bool  `json:"matches"`
}

type Service struct {
	db *sql.DB
}

func NewService(database *sql.DB) *Service {
	return &Service{db: database}
}

func (s *Service) ListWallets(statusFilter string) ([]AdminWallet, error) {
	query := `
		SELECT w.id, w.user_id, u.name, u.email, u.handle, u.kyc_level,
		       w.balance_paise, w.status, w.daily_send_limit_paise, w.created_at
		FROM wallets w
		JOIN users u ON w.user_id = u.id
	`
	var args []any
	if statusFilter != "" {
		query += " WHERE w.status = ?"
		args = append(args, statusFilter)
	}
	query += " ORDER BY w.created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wallets []AdminWallet
	for rows.Next() {
		var aw AdminWallet
		err := rows.Scan(
			&aw.ID, &aw.UserID, &aw.UserName, &aw.UserEmail, &aw.UserHandle, &aw.KYCLevel,
			&aw.BalancePaise, &aw.Status, &aw.DailySendLimitPaise, &aw.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		wallets = append(wallets, aw)
	}

	if wallets == nil {
		wallets = []AdminWallet{}
	}

	return wallets, nil
}

func (s *Service) GetWalletDetail(walletID string) (*AdminWalletDetail, error) {
	var aw AdminWallet
	err := s.db.QueryRow(`
		SELECT w.id, w.user_id, u.name, u.email, u.handle, u.kyc_level,
		       w.balance_paise, w.status, w.daily_send_limit_paise, w.created_at
		FROM wallets w
		JOIN users u ON w.user_id = u.id
		WHERE w.id = ?
	`, walletID).Scan(
		&aw.ID, &aw.UserID, &aw.UserName, &aw.UserEmail, &aw.UserHandle, &aw.KYCLevel,
		&aw.BalancePaise, &aw.Status, &aw.DailySendLimitPaise, &aw.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &httpx.CustomError{Code: "wallet_not_found", Message: "Wallet not found", Status: http.StatusNotFound}
		}
		return nil, err
	}

	rows, err := s.db.Query(`
		SELECT t.id, t.wallet_id, t.type, t.amount_paise, t.balance_after_paise, t.reference_id,
		       t.counterparty_wallet_id, u.name, u.handle, t.note, t.created_at
		FROM transactions t
		LEFT JOIN wallets cw ON t.counterparty_wallet_id = cw.id
		LEFT JOIN users u ON cw.user_id = u.id
		WHERE t.wallet_id = ?
		ORDER BY t.created_at DESC
		LIMIT 100
	`, walletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txns []wallet.Transaction
	for rows.Next() {
		var t wallet.Transaction
		var cpWalletID, cpName, cpHandle sql.NullString
		err := rows.Scan(
			&t.ID, &t.WalletID, &t.Type, &t.AmountPaise, &t.BalanceAfterPaise, &t.ReferenceID,
			&cpWalletID, &cpName, &cpHandle, &t.Note, &t.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if cpWalletID.Valid {
			t.CounterpartyWalletID = &cpWalletID.String
		}
		if cpName.Valid {
			t.CounterpartyName = &cpName.String
		}
		if cpHandle.Valid {
			t.CounterpartyHandle = &cpHandle.String
		}
		txns = append(txns, t)
	}

	if txns == nil {
		txns = []wallet.Transaction{}
	}

	return &AdminWalletDetail{
		Wallet:       aw,
		Transactions: txns,
	}, nil
}

func (s *Service) SetWalletStatus(walletID string, status string) error {
	res, err := s.db.Exec("UPDATE wallets SET status = ? WHERE id = ?", status, walletID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return &httpx.CustomError{Code: "wallet_not_found", Message: "Wallet not found", Status: http.StatusNotFound}
	}
	return nil
}

func (s *Service) ListTransfers(statusFilter string) ([]any, error) {
	query := `
		SELECT t.id, t.from_wallet_id, fu.name, fu.handle, t.to_wallet_id, tu.name, tu.handle,
		       t.amount_paise, t.note, t.status, t.failure_reason, t.created_at
		FROM transfers t
		JOIN wallets fw ON t.from_wallet_id = fw.id
		JOIN users fu ON fw.user_id = fu.id
		JOIN wallets tw ON t.to_wallet_id = tw.id
		JOIN users tu ON tw.user_id = tu.id
	`
	var args []any
	if statusFilter != "" {
		query += " WHERE t.status = ?"
		args = append(args, statusFilter)
	}
	query += " ORDER BY t.created_at DESC LIMIT 200"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []any
	for rows.Next() {
		var tr struct {
			ID            string    `json:"id"`
			FromWalletID  string    `json:"from_wallet_id"`
			FromUserName  string    `json:"from_user_name"`
			FromUserHandle string   `json:"from_user_handle"`
			ToWalletID    string    `json:"to_wallet_id"`
			ToUserName    string    `json:"to_user_name"`
			ToUserHandle  string    `json:"to_user_handle"`
			AmountPaise   int64     `json:"amount_paise"`
			Note          string    `json:"note"`
			Status        string    `json:"status"`
			FailureReason *string   `json:"failure_reason,omitempty"`
			CreatedAt     time.Time `json:"created_at"`
		}
		var failReason sql.NullString
		err := rows.Scan(
			&tr.ID, &tr.FromWalletID, &tr.FromUserName, &tr.FromUserHandle,
			&tr.ToWalletID, &tr.ToUserName, &tr.ToUserHandle,
			&tr.AmountPaise, &tr.Note, &tr.Status, &failReason, &tr.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if failReason.Valid {
			tr.FailureReason = &failReason.String
		}
		result = append(result, tr)
	}

	if result == nil {
		result = []any{}
	}

	return result, nil
}

func (s *Service) GetMetrics() (*AdminMetrics, error) {
	var totalHeld int64
	_ = s.db.QueryRow("SELECT COALESCE(SUM(balance_paise), 0) FROM wallets").Scan(&totalHeld)

	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)
	var todayVolume int64
	_ = s.db.QueryRow("SELECT COALESCE(SUM(amount_paise), 0) FROM transfers WHERE status = 'completed' AND created_at >= ?", startOfDay).Scan(&todayVolume)

	var totalCount, failedCount int64
	_ = s.db.QueryRow("SELECT COUNT(1) FROM transfers").Scan(&totalCount)
	_ = s.db.QueryRow("SELECT COUNT(1) FROM transfers WHERE status = 'failed'").Scan(&failedCount)

	rate := 0.0
	if totalCount > 0 {
		rate = float64(failedCount) / float64(totalCount)
	}

	rows, err := s.db.Query(`
		SELECT u.id, u.name, u.handle, SUM(t.amount_paise) as total_paise, COUNT(1) as cnt
		FROM transfers t
		JOIN wallets w ON t.from_wallet_id = w.id
		JOIN users u ON w.user_id = u.id
		WHERE t.status = 'completed'
		GROUP BY u.id
		ORDER BY total_paise DESC
		LIMIT 5
	`)
	var top []TopSender
	if err == nil {
		for rows.Next() {
			var ts TopSender
			if err := rows.Scan(&ts.UserID, &ts.UserName, &ts.UserHandle, &ts.TotalPaise, &ts.Count); err == nil {
				top = append(top, ts)
			}
		}
		rows.Close()
	}
	if top == nil {
		top = []TopSender{}
	}

	return &AdminMetrics{
		TotalBalancesHeldPaise: totalHeld,
		TodayVolumePaise:       todayVolume,
		TotalTransfersCount:    totalCount,
		FailedTransferRate:     rate,
		TopSenders:             top,
	}, nil
}

func (s *Service) CheckLedger() (*LedgerCheckResponse, error) {
	var sumBalances, sumLedger int64
	_ = s.db.QueryRow("SELECT COALESCE(SUM(balance_paise), 0) FROM wallets").Scan(&sumBalances)
	_ = s.db.QueryRow("SELECT COALESCE(SUM(amount_paise), 0) FROM transactions").Scan(&sumLedger)

	return &LedgerCheckResponse{
		SumOfBalances: sumBalances,
		SumOfLedger:   sumLedger,
		Matches:       sumBalances == sumLedger,
	}, nil
}
