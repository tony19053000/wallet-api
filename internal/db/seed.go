package db

import (
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type SeedUser struct {
	Name         string
	Email        string
	Phone        string
	Handle       string
	Password     string
	Role         string
	KYCLevel     string
	BalancePaise int64
	Status       string
	BankName     string
	BankHolder   string
	BankAcc      string
	IFSC         string
}

func Seed(database *sql.DB, adminEmail, adminPassword string) error {
	var count int
	err := database.QueryRow("SELECT COUNT(1) FROM users WHERE email = ?", adminEmail).Scan(&count)
	if err == nil && count > 0 {
		return nil // Already seeded
	}

	seedUsers := []SeedUser{
		{
			Name:         "Riya Sharma",
			Email:        "riya@pocketa.money",
			Phone:        "+91 98201 44521",
			Handle:       "@riya.s",
			Password:     "riya-2026",
			Role:         "user",
			KYCLevel:     "full",
			BalancePaise: 2450000, // ₹24,500.00
			Status:       "active",
			BankName:     "HDFC Bank",
			BankHolder:   "Riya Sharma",
			BankAcc:      "501004821948",
			IFSC:         "HDFC0000240",
		},
		{
			Name:         "Arjun Mehta",
			Email:        "arjun@pocketa.money",
			Phone:        "+91 98110 88234",
			Handle:       "@arjun.m",
			Password:     "arjun-2026",
			Role:         "user",
			KYCLevel:     "full",
			BalancePaise: 4830000, // ₹48,300.00
			Status:       "active",
			BankName:     "ICICI Bank",
			BankHolder:   "Arjun Mehta",
			BankAcc:      "000401583920",
			IFSC:         "ICIC0000004",
		},
		{
			Name:         "Risk & Ops Admin",
			Email:        adminEmail,
			Phone:        "+91 99999 00000",
			Handle:       "@risk.ops",
			Password:     adminPassword,
			Role:         "admin",
			KYCLevel:     "full",
			BalancePaise: 10000000, // ₹1,00,000.00
			Status:       "active",
			BankName:     "State Bank of India",
			BankHolder:   "Risk & Ops Admin",
			BankAcc:      "30948201948",
			IFSC:         "SBIN0000691",
		},
		{
			Name:         "Ananya Roy",
			Email:        "ananya@pocketa.money",
			Phone:        "+91 97412 33491",
			Handle:       "@ananya.r",
			Password:     "pocketa-2026",
			Role:         "user",
			KYCLevel:     "full",
			BalancePaise: 1845000, // ₹18,450.00
			Status:       "active",
			BankName:     "Axis Bank",
			BankHolder:   "Ananya Roy",
			BankAcc:      "918020048201",
			IFSC:         "UTIB0000010",
		},
		{
			Name:         "Kabir Verma",
			Email:        "kabir@pocketa.money",
			Phone:        "+91 98765 12340",
			Handle:       "@kabir.v",
			Password:     "pocketa-2026",
			Role:         "user",
			KYCLevel:     "basic",
			BalancePaise: 560000, // ₹5,600.00
			Status:       "active",
			BankName:     "Kotak Mahindra Bank",
			BankHolder:   "Kabir Verma",
			BankAcc:      "8412093841",
			IFSC:         "KKBK0000181",
		},
		{
			Name:         "Priya Patel",
			Email:        "priya@pocketa.money",
			Phone:        "+91 99001 88722",
			Handle:       "@priya.p",
			Password:     "pocketa-2026",
			Role:         "user",
			KYCLevel:     "full",
			BalancePaise: 3210000, // ₹32,100.00
			Status:       "active",
			BankName:     "HDFC Bank",
			BankHolder:   "Priya Patel",
			BankAcc:      "501009988221",
			IFSC:         "HDFC0000240",
		},
		{
			Name:         "Rohan Gupta",
			Email:        "rohan@pocketa.money",
			Phone:        "+91 98450 77112",
			Handle:       "@rohan.g",
			Password:     "pocketa-2026",
			Role:         "user",
			KYCLevel:     "basic",
			BalancePaise: 12000, // ₹120.00
			Status:       "active",
			BankName:     "State Bank of India",
			BankHolder:   "Rohan Gupta",
			BankAcc:      "20194820193",
			IFSC:         "SBIN0000691",
		},
		{
			Name:         "Sneha Reddy",
			Email:        "sneha@pocketa.money",
			Phone:        "+91 98860 44109",
			Handle:       "@sneha.r",
			Password:     "pocketa-2026",
			Role:         "user",
			KYCLevel:     "full",
			BalancePaise: 1280000, // ₹12,800.00
			Status:       "frozen",
			BankName:     "ICICI Bank",
			BankHolder:   "Sneha Reddy",
			BankAcc:      "000409981240",
			IFSC:         "ICIC0000004",
		},
		{
			Name:         "Vikram Singh",
			Email:        "vikram@pocketa.money",
			Phone:        "+91 97112 55901",
			Handle:       "@vikram.s",
			Password:     "pocketa-2026",
			Role:         "user",
			KYCLevel:     "basic",
			BalancePaise: 895000, // ₹8,950.00
			Status:       "active",
			BankName:     "Axis Bank",
			BankHolder:   "Vikram Singh",
			BankAcc:      "918044332211",
			IFSC:         "UTIB0000010",
		},
		{
			Name:         "Diya Nair",
			Email:        "diya@pocketa.money",
			Phone:        "+91 99451 22384",
			Handle:       "@diya.n",
			Password:     "pocketa-2026",
			Role:         "user",
			KYCLevel:     "full",
			BalancePaise: 2940000, // ₹29,400.00
			Status:       "active",
			BankName:     "Kotak Mahindra Bank",
			BankHolder:   "Diya Nair",
			BankAcc:      "6712094833",
			IFSC:         "KKBK0000181",
		},
		{
			Name:         "Aditya Iyer",
			Email:        "aditya@pocketa.money",
			Phone:        "+91 98231 66702",
			Handle:       "@aditya.i",
			Password:     "pocketa-2026",
			Role:         "user",
			KYCLevel:     "basic",
			BalancePaise: 320000, // ₹3,200.00
			Status:       "active",
			BankName:     "HDFC Bank",
			BankHolder:   "Aditya Iyer",
			BankAcc:      "501007788112",
			IFSC:         "HDFC0000240",
		},
		{
			Name:         "Meera Joshi",
			Email:        "meera@pocketa.money",
			Phone:        "+91 97654 88120",
			Handle:       "@meera.j",
			Password:     "pocketa-2026",
			Role:         "user",
			KYCLevel:     "full",
			BalancePaise: 1575000, // ₹15,750.00
			Status:       "active",
			BankName:     "ICICI Bank",
			BankHolder:   "Meera Joshi",
			BankAcc:      "000405544332",
			IFSC:         "ICIC0000004",
		},
		{
			Name:         "Siddharth Rao",
			Email:        "siddharth@pocketa.money",
			Phone:        "+91 98199 44321",
			Handle:       "@sid.rao",
			Password:     "pocketa-2026",
			Role:         "user",
			KYCLevel:     "basic",
			BalancePaise: 94000, // ₹940.00
			Status:       "active",
			BankName:     "State Bank of India",
			BankHolder:   "Siddharth Rao",
			BankAcc:      "30887766554",
			IFSC:         "SBIN0000691",
		},
		{
			Name:         "Tara Kapoor",
			Email:        "tara@pocketa.money",
			Phone:        "+91 99200 11984",
			Handle:       "@tara.k",
			Password:     "pocketa-2026",
			Role:         "user",
			KYCLevel:     "full",
			BalancePaise: 4120000, // ₹41,200.00
			Status:       "active",
			BankName:     "Axis Bank",
			BankHolder:   "Tara Kapoor",
			BankAcc:      "918011223344",
			IFSC:         "UTIB0000010",
		},
	}

	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().UTC()

	type createdUserRecord struct {
		user_id   string
		wallet_id string
		bank_id   string
		handle    string
	}
	userRecords := make(map[string]createdUserRecord)

	for _, su := range seedUsers {
		userID := NewID("usr_")
		walletID := NewID("wal_")
		bankID := NewID("bnk_")

		hash, err := bcrypt.GenerateFromPassword([]byte(su.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		limit := int64(1000000) // basic ₹10,000
		if su.KYCLevel == "full" {
			limit = int64(10000000) // full ₹1,00,000
		}

		createdTime := now.AddDate(0, 0, -90)

		_, err = tx.Exec(`
			INSERT INTO users (id, name, email, phone, handle, password_hash, role, kyc_level, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, userID, su.Name, su.Email, su.Phone, su.Handle, string(hash), su.Role, su.KYCLevel, createdTime)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			INSERT INTO wallets (id, user_id, balance_paise, status, daily_send_limit_paise, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, walletID, userID, su.BalancePaise, su.Status, limit, createdTime)
		if err != nil {
			return err
		}

		last4 := su.BankAcc[len(su.BankAcc)-4:]
		_, err = tx.Exec(`
			INSERT INTO bank_accounts (id, user_id, bank_name, account_holder, account_last4, ifsc, is_primary, created_at)
			VALUES (?, ?, ?, ?, ?, ?, 1, ?)
		`, bankID, userID, su.BankName, su.BankHolder, last4, su.IFSC, createdTime)
		if err != nil {
			return err
		}

		userRecords[su.Handle] = createdUserRecord{
			user_id:   userID,
			wallet_id: walletID,
			bank_id:   bankID,
			handle:    su.Handle,
		}
	}

	// Generate 90 days of transactions, transfers, requests, and withdrawals
	notes := []string{
		"Goa trip 🏖️ fuel",
		"Rent share — Oct",
		"Swiggy dinner split",
		"Chai ☕",
		"Birthday gift for Ananya",
		"Weekend brunch 🥐",
		"Movie tickets 🍿",
		"Uber cab share 🚕",
		"Concert pass 🎸",
		"Electricity bill split ⚡",
		"Groceries 🛒",
		"Zomato pizza night 🍕",
	}

	// 1. Initial Top-ups for all users
	for _, rec := range userRecords {
		topUpAmt := int64(5000000) // ₹50,000 top up initially
		txnID := NewID("txn_")
		refID := NewID("ref_")
		topUpTime := now.AddDate(0, 0, -88)

		_, err = tx.Exec(`
			INSERT INTO transactions (id, wallet_id, type, amount_paise, balance_after_paise, reference_id, counterparty_wallet_id, note, created_at)
			VALUES (?, ?, 'top_up', ?, ?, ?, NULL, 'Initial wallet top-up via UPI', ?)
		`, txnID, rec.wallet_id, topUpAmt, topUpAmt, refID, topUpTime)
		if err != nil {
			return err
		}
	}

	// 2. Realistic transfers between users over the past 85 days
	handles := []string{"@riya.s", "@arjun.m", "@ananya.r", "@kabir.v", "@priya.p", "@rohan.g", "@vikram.s", "@diya.n", "@tara.k"}
	for i := 85; i >= 1; i-- {
		fromH := handles[i%len(handles)]
		toH := handles[(i+3)%len(handles)]
		if fromH == toH {
			continue
		}

		fromRec := userRecords[fromH]
		toRec := userRecords[toH]
		note := notes[i%len(notes)]
		amount := int64((150 + (i * 35) % 4500) * 100) // ₹150 to ₹4,650
		trfTime := now.AddDate(0, 0, -i).Add(time.Duration(i*3) * time.Hour)

		trfID := NewID("trf_")
		refOutID := NewID("ref_")
		refInID := NewID("ref_")
		txnOutID := NewID("txn_")
		txnInID := NewID("txn_")

		// Create completed transfer
		_, err = tx.Exec(`
			INSERT INTO transfers (id, from_wallet_id, to_wallet_id, amount_paise, note, status, failure_reason, created_at)
			VALUES (?, ?, ?, ?, ?, 'completed', NULL, ?)
		`, trfID, fromRec.wallet_id, toRec.wallet_id, amount, note, trfTime)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			INSERT INTO transactions (id, wallet_id, type, amount_paise, balance_after_paise, reference_id, counterparty_wallet_id, note, created_at)
			VALUES (?, ?, 'transfer_out', ?, 2500000, ?, ?, ?, ?)
		`, txnOutID, fromRec.wallet_id, -amount, refOutID, toRec.wallet_id, note, trfTime)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			INSERT INTO transactions (id, wallet_id, type, amount_paise, balance_after_paise, reference_id, counterparty_wallet_id, note, created_at)
			VALUES (?, ?, 'transfer_in', ?, 3500000, ?, ?, ?, ?)
		`, txnInID, toRec.wallet_id, amount, refInID, fromRec.wallet_id, note, trfTime)
		if err != nil {
			return err
		}
	}

	// 3. Money requests in various statuses (pending, paid, declined, cancelled)
	statuses := []string{"pending", "paid", "declined", "cancelled"}
	for i := 0; i < 20; i++ {
		fromH := handles[i%len(handles)]
		toH := handles[(i+2)%len(handles)]
		if fromH == toH {
			continue
		}

		fromRec := userRecords[fromH]
		toRec := userRecords[toH]
		st := statuses[i%len(statuses)]
		reqAmt := int64((200 + i*150) * 100)
		reqTime := now.AddDate(0, 0, -(i * 4))

		reqID := NewID("req_")
		note := fmt.Sprintf("Split request for %s", notes[i%len(notes)])

		_, err = tx.Exec(`
			INSERT INTO money_requests (id, from_user_id, to_user_id, amount_paise, note, status, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, reqID, fromRec.user_id, toRec.user_id, reqAmt, note, st, reqTime)
		if err != nil {
			return err
		}
	}

	// 4. Sample withdrawals to bank accounts
	for i, h := range []string{"@riya.s", "@arjun.m", "@priya.p", "@tara.k", "@diya.n"} {
		rec := userRecords[h]
		wdrAmt := int64(1000000) // ₹10,000
		feeAmt := int64(200)     // ₹2
		wdrTime := now.AddDate(0, 0, -(i*10 + 2))
		wdrID := NewID("wdr_")
		txnWdrID := NewID("txn_")
		txnFeeID := NewID("txn_")
		refWdrID := NewID("ref_")
		refFeeID := NewID("ref_")

		_, err = tx.Exec(`
			INSERT INTO withdrawals (id, wallet_id, bank_account_id, amount_paise, fee_paise, status, created_at)
			VALUES (?, ?, ?, ?, ?, 'completed', ?)
		`, wdrID, rec.wallet_id, rec.bank_id, wdrAmt, feeAmt, wdrTime)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			INSERT INTO transactions (id, wallet_id, type, amount_paise, balance_after_paise, reference_id, counterparty_wallet_id, note, created_at)
			VALUES (?, ?, 'withdrawal', ?, 2000000, ?, NULL, 'Bank withdrawal', ?)
		`, txnWdrID, rec.wallet_id, -wdrAmt, refWdrID, wdrTime)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			INSERT INTO transactions (id, wallet_id, type, amount_paise, balance_after_paise, reference_id, counterparty_wallet_id, note, created_at)
			VALUES (?, ?, 'fee', ?, 1999800, ?, NULL, 'Withdrawal fee', ?)
		`, txnFeeID, rec.wallet_id, -feeAmt, refFeeID, wdrTime)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
