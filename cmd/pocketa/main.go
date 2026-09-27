package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tony19053000/wallet-api/internal/admin"
	"github.com/tony19053000/wallet-api/internal/auth"
	"github.com/tony19053000/wallet-api/internal/bank"
	"github.com/tony19053000/wallet-api/internal/config"
	"github.com/tony19053000/wallet-api/internal/db"
	"github.com/tony19053000/wallet-api/internal/httpx"
	"github.com/tony19053000/wallet-api/internal/request"
	"github.com/tony19053000/wallet-api/internal/transfer"
	"github.com/tony19053000/wallet-api/internal/wallet"
	"github.com/tony19053000/wallet-api/internal/web"
	"github.com/tony19053000/wallet-api/internal/withdrawal"
	webfs "github.com/tony19053000/wallet-api/web"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	logger.Info("starting pocketa server",
		slog.String("port", cfg.Port),
		slog.String("host", cfg.Host),
		slog.String("db_path", cfg.DatabasePath),
	)

	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		logger.Error("failed to open database", slog.Any("error", err))
		os.Exit(1)
	}
	defer database.Close()

	if cfg.SeedData {
		if err := db.Seed(database, cfg.AdminEmail, cfg.AdminPassword); err != nil {
			logger.Error("failed to seed database", slog.Any("error", err))
		} else {
			logger.Info("database seeding completed")
		}
	}

	// Initialize Services
	authSvc := auth.NewService(database)
	walletSvc := wallet.NewService(database)
	transferSvc := transfer.NewService(database)
	requestSvc := request.NewService(database, transferSvc)
	bankSvc := bank.NewService(database)
	wdrSvc := withdrawal.NewService(database)
	adminSvc := admin.NewService(database)

	// Initialize Handlers
	authH := auth.NewHandler(authSvc)
	walletH := wallet.NewHandler(walletSvc)
	transferH := transfer.NewHandler(transferSvc)
	requestH := request.NewHandler(requestSvc)
	bankH := bank.NewHandler(bankSvc)
	wdrH := withdrawal.NewHandler(wdrSvc)
	adminH := admin.NewHandler(adminSvc)
	webH := web.NewHandler(authSvc, walletSvc, transferSvc, requestSvc, bankSvc, wdrSvc, adminSvc)

	mux := http.NewServeMux()

	// Health check endpoint (No Auth)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Static files serving
	staticFS, err := fs.Sub(webfs.FS, "static")
	if err == nil {
		mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	}

	// Public Web Pages
	mux.HandleFunc("GET /{$}", webH.Index)
	mux.HandleFunc("GET /fees", webH.Fees)
	mux.HandleFunc("GET /security", webH.Security)
	mux.HandleFunc("GET /about", webH.About)
	mux.HandleFunc("GET /careers", webH.Careers)
	mux.HandleFunc("GET /help", webH.Help)
	mux.HandleFunc("GET /contact", webH.Contact)
	mux.HandleFunc("GET /terms", webH.Terms)
	mux.HandleFunc("GET /privacy", webH.Privacy)
	mux.HandleFunc("GET /grievance", webH.Grievance)
	mux.HandleFunc("GET /login", webH.Login)
	mux.HandleFunc("GET /signup", webH.Signup)
	mux.HandleFunc("GET /forgot", webH.Forgot)

	// App Web Pages
	mux.HandleFunc("GET /app", webH.AppHome)
	mux.HandleFunc("GET /app/add", webH.AppAdd)
	mux.HandleFunc("GET /app/send", webH.AppSend)
	mux.HandleFunc("GET /app/transfers/{id}", webH.AppReceipt)
	mux.HandleFunc("GET /app/request", webH.AppRequest)
	mux.HandleFunc("GET /app/requests", webH.AppRequestsList)
	mux.HandleFunc("GET /app/activity", webH.AppActivity)
	mux.HandleFunc("GET /app/withdraw", webH.AppWithdraw)
	mux.HandleFunc("GET /app/banks", webH.AppBanks)
	mux.HandleFunc("GET /app/profile", webH.AppProfile)

	// Admin Web Pages
	mux.HandleFunc("GET /admin", webH.AdminDashboard)
	mux.HandleFunc("GET /admin/wallets", webH.AdminWallets)
	mux.HandleFunc("GET /admin/wallets/{id}", webH.AdminWalletDetail)
	mux.HandleFunc("GET /admin/transfers", webH.AdminTransfers)
	mux.HandleFunc("GET /admin/reconciliation", webH.AdminReconciliation)

	// REST API v1 Routes
	// Auth
	mux.HandleFunc("POST /api/v1/auth/signup", authH.Signup)
	mux.HandleFunc("POST /api/v1/auth/login", authH.Login)
	mux.HandleFunc("POST /api/v1/auth/logout", authH.Logout)
	mux.HandleFunc("GET /api/v1/me", authH.Me)

	// Wallet
	mux.HandleFunc("GET /api/v1/wallet", authH.AuthMiddleware(walletH.GetWallet))
	mux.HandleFunc("POST /api/v1/wallet/top-up", authH.AuthMiddleware(walletH.TopUp))
	mux.HandleFunc("GET /api/v1/wallet/transactions", authH.AuthMiddleware(walletH.GetTransactions))
	mux.HandleFunc("GET /api/v1/wallet/statement.csv", authH.AuthMiddleware(walletH.GetStatementCSV))

	// Transfers
	mux.HandleFunc("GET /api/v1/users/lookup", transferH.LookupUser)
	mux.HandleFunc("POST /api/v1/transfers", authH.AuthMiddleware(transferH.Transfer))
	mux.HandleFunc("GET /api/v1/transfers", authH.AuthMiddleware(transferH.ListTransfers))
	mux.HandleFunc("GET /api/v1/transfers/{id}", authH.AuthMiddleware(transferH.GetTransfer))

	// Requests & Split
	mux.HandleFunc("POST /api/v1/requests", authH.AuthMiddleware(requestH.CreateRequest))
	mux.HandleFunc("GET /api/v1/requests", authH.AuthMiddleware(requestH.ListRequests))
	mux.HandleFunc("POST /api/v1/requests/{id}/pay", authH.AuthMiddleware(requestH.PayRequest))
	mux.HandleFunc("POST /api/v1/requests/{id}/decline", authH.AuthMiddleware(requestH.DeclineRequest))
	mux.HandleFunc("POST /api/v1/requests/{id}/cancel", authH.AuthMiddleware(requestH.CancelRequest))
	mux.HandleFunc("POST /api/v1/split", authH.AuthMiddleware(requestH.SplitBill))

	// Bank Accounts & Withdrawals
	mux.HandleFunc("GET /api/v1/bank-accounts", authH.AuthMiddleware(bankH.ListBankAccounts))
	mux.HandleFunc("POST /api/v1/bank-accounts", authH.AuthMiddleware(bankH.AddBankAccount))
	mux.HandleFunc("DELETE /api/v1/bank-accounts/{id}", authH.AuthMiddleware(bankH.DeleteBankAccount))
	mux.HandleFunc("POST /api/v1/withdrawals", authH.AuthMiddleware(wdrH.Withdraw))
	mux.HandleFunc("GET /api/v1/withdrawals", authH.AuthMiddleware(wdrH.ListWithdrawals))

	// Admin API
	mux.HandleFunc("GET /api/v1/admin/wallets", authH.AdminMiddleware(adminH.ListWallets))
	mux.HandleFunc("GET /api/v1/admin/wallets/{id}", authH.AdminMiddleware(adminH.GetWallet))
	mux.HandleFunc("POST /api/v1/admin/wallets/{id}/freeze", authH.AdminMiddleware(adminH.FreezeWallet))
	mux.HandleFunc("POST /api/v1/admin/wallets/{id}/unfreeze", authH.AdminMiddleware(adminH.UnfreezeWallet))
	mux.HandleFunc("GET /api/v1/admin/transfers", authH.AdminMiddleware(adminH.ListTransfers))
	mux.HandleFunc("GET /api/v1/admin/metrics", authH.AdminMiddleware(adminH.GetMetrics))
	mux.HandleFunc("GET /api/v1/admin/ledger/check", authH.AdminMiddleware(adminH.CheckLedger))

	// Apply Middlewares
	handler := httpx.RecoveryMiddleware(logger)(httpx.LoggingMiddleware(logger)(mux))

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info(fmt.Sprintf("server listening on %s", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server listener error", slog.Any("error", err))
		}
	}()

	<-stop
	logger.Info("shutting down server gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown forced", slog.Any("error", err))
	} else {
		logger.Info("server stopped gracefully")
	}
}
