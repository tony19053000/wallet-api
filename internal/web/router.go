package web

import (
	"html/template"
	"net/http"

	"github.com/tony19053000/wallet-api/internal/admin"
	"github.com/tony19053000/wallet-api/internal/auth"
	"github.com/tony19053000/wallet-api/internal/bank"
	"github.com/tony19053000/wallet-api/internal/httpx"
	"github.com/tony19053000/wallet-api/internal/request"
	"github.com/tony19053000/wallet-api/internal/transfer"
	"github.com/tony19053000/wallet-api/internal/wallet"
	"github.com/tony19053000/wallet-api/internal/withdrawal"
	"github.com/tony19053000/wallet-api/web"
)

type Handler struct {
	authSvc     *auth.Service
	walletSvc   *wallet.Service
	transferSvc *transfer.Service
	requestSvc  *request.Service
	bankSvc     *bank.Service
	wdrSvc      *withdrawal.Service
	adminSvc    *admin.Service
}

func NewHandler(
	authSvc *auth.Service,
	walletSvc *wallet.Service,
	transferSvc *transfer.Service,
	requestSvc *request.Service,
	bankSvc *bank.Service,
	wdrSvc *withdrawal.Service,
	adminSvc *admin.Service,
) *Handler {
	return &Handler{
		authSvc:     authSvc,
		walletSvc:   walletSvc,
		transferSvc: transferSvc,
		requestSvc:  requestSvc,
		bankSvc:     bankSvc,
		wdrSvc:      wdrSvc,
		adminSvc:    adminSvc,
	}
}

func (h *Handler) renderPublic(w http.ResponseWriter, r *http.Request, pageName, title string, data map[string]any) {
	if data == nil {
		data = make(map[string]any)
	}
	data["Title"] = title

	token := httpx.ExtractToken(r)
	if u, err := h.authSvc.GetUserByToken(token); err == nil {
		data["User"] = u
	}

	tmpl, err := template.ParseFS(web.FS, "templates/layout_public.html", "templates/"+pageName+".html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, "layout_public", data)
}

func (h *Handler) renderApp(w http.ResponseWriter, r *http.Request, pageName, title, activePage string, data map[string]any) {
	token := httpx.ExtractToken(r)
	u, err := h.authSvc.GetUserByToken(token)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if data == nil {
		data = make(map[string]any)
	}
	data["Title"] = title
	data["Page"] = activePage
	data["User"] = u

	tmpl, err := template.ParseFS(web.FS, "templates/layout_app.html", "templates/"+pageName+".html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, "layout_app", data)
}

func (h *Handler) renderAdmin(w http.ResponseWriter, r *http.Request, pageName, title, activePage string, data map[string]any) {
	token := httpx.ExtractToken(r)
	u, err := h.authSvc.GetUserByToken(token)
	if err != nil || u.Role != "admin" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if data == nil {
		data = make(map[string]any)
	}
	data["Title"] = title
	data["Page"] = activePage
	data["User"] = u

	tmpl, err := template.ParseFS(web.FS, "templates/layout_admin.html", "templates/"+pageName+".html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, "layout_admin", data)
}

// Public Page Handlers
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	h.renderPublic(w, r, "index", "Home", nil)
}
func (h *Handler) Fees(w http.ResponseWriter, r *http.Request) {
	h.renderPublic(w, r, "fees", "Fees & Limits", nil)
}
func (h *Handler) Security(w http.ResponseWriter, r *http.Request) {
	h.renderPublic(w, r, "security", "Security Architecture", nil)
}
func (h *Handler) About(w http.ResponseWriter, r *http.Request) {
	h.renderPublic(w, r, "about", "About Us", nil)
}
func (h *Handler) Careers(w http.ResponseWriter, r *http.Request) {
	h.renderPublic(w, r, "careers", "Careers", nil)
}
func (h *Handler) Help(w http.ResponseWriter, r *http.Request) {
	h.renderPublic(w, r, "help", "Help Centre", nil)
}
func (h *Handler) Contact(w http.ResponseWriter, r *http.Request) {
	h.renderPublic(w, r, "contact", "Contact Support", nil)
}
func (h *Handler) Terms(w http.ResponseWriter, r *http.Request) {
	h.renderPublic(w, r, "terms", "Terms of Service", nil)
}
func (h *Handler) Privacy(w http.ResponseWriter, r *http.Request) {
	h.renderPublic(w, r, "privacy", "Privacy Policy", nil)
}
func (h *Handler) Grievance(w http.ResponseWriter, r *http.Request) {
	h.renderPublic(w, r, "grievance", "Grievance Redressal", nil)
}
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	h.renderPublic(w, r, "login", "Log In", nil)
}
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	h.renderPublic(w, r, "signup", "Sign Up", nil)
}
func (h *Handler) Forgot(w http.ResponseWriter, r *http.Request) {
	h.renderPublic(w, r, "forgot", "Forgot Password", nil)
}

// App Page Handlers
func (h *Handler) AppHome(w http.ResponseWriter, r *http.Request) {
	token := httpx.ExtractToken(r)
	u, err := h.authSvc.GetUserByToken(token)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	wRes, _ := h.walletSvc.GetWalletResponse(u.ID)
	txns, _ := h.walletSvc.GetTransactions(u.ID, wallet.TransactionFilter{Limit: 5})

	type formattedTxn struct {
		wallet.Transaction
		FormattedAmount string
	}
	var fmtTxns []formattedTxn
	for _, t := range txns {
		fmtTxns = append(fmtTxns, formattedTxn{
			Transaction:     t,
			FormattedAmount: httpx.FormatPaise(t.AmountPaise),
		})
	}

	balStr := "₹0.00"
	if wRes != nil {
		balStr = httpx.FormatPaise(wRes.BalancePaise)
	}

	h.renderApp(w, r, "app_home", "Dashboard", "home", map[string]any{
		"Wallet":             wRes,
		"FormattedBalance":   balStr,
		"RecentTransactions": fmtTxns,
	})
}

func (h *Handler) AppAdd(w http.ResponseWriter, r *http.Request) {
	h.renderApp(w, r, "app_add", "Top Up Wallet", "home", nil)
}

func (h *Handler) AppSend(w http.ResponseWriter, r *http.Request) {
	h.renderApp(w, r, "app_send", "Send Money", "send", nil)
}

func (h *Handler) AppReceipt(w http.ResponseWriter, r *http.Request) {
	token := httpx.ExtractToken(r)
	u, err := h.authSvc.GetUserByToken(token)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	id := r.PathValue("id")
	tr, err := h.transferSvc.GetTransferByID(id, u.ID)
	if err != nil {
		http.Redirect(w, r, "/app", http.StatusSeeOther)
		return
	}

	h.renderApp(w, r, "app_receipt", "Transfer Receipt", "send", map[string]any{
		"Transfer":        tr,
		"FormattedAmount": httpx.FormatPaise(tr.AmountPaise),
	})
}

func (h *Handler) AppRequest(w http.ResponseWriter, r *http.Request) {
	h.renderApp(w, r, "app_request", "Request & Split", "request", nil)
}

func (h *Handler) AppRequestsList(w http.ResponseWriter, r *http.Request) {
	token := httpx.ExtractToken(r)
	u, err := h.authSvc.GetUserByToken(token)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	dir := r.URL.Query().Get("direction")
	if dir == "" {
		dir = "incoming"
	}

	reqs, _ := h.requestSvc.ListRequests(u.ID, dir)

	type formattedReq struct {
		request.MoneyRequest
		FormattedAmount string
	}
	var fmtReqs []formattedReq
	for _, req := range reqs {
		fmtReqs = append(fmtReqs, formattedReq{
			MoneyRequest:    req,
			FormattedAmount: httpx.FormatPaise(req.AmountPaise),
		})
	}

	h.renderApp(w, r, "app_requests", "Money Requests", "requests", map[string]any{
		"Requests":  fmtReqs,
		"Direction": dir,
	})
}

func (h *Handler) AppActivity(w http.ResponseWriter, r *http.Request) {
	token := httpx.ExtractToken(r)
	u, err := h.authSvc.GetUserByToken(token)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	query := r.URL.Query()
	fType := query.Get("type")
	fFrom := query.Get("from")
	fTo := query.Get("to")

	txns, _ := h.walletSvc.GetTransactions(u.ID, wallet.TransactionFilter{
		Type:  fType,
		From:  fFrom,
		To:    fTo,
		Limit: 100,
	})

	type formattedTxn struct {
		wallet.Transaction
		FormattedAmount       string
		FormattedBalanceAfter string
	}
	var fmtTxns []formattedTxn
	for _, t := range txns {
		fmtTxns = append(fmtTxns, formattedTxn{
			Transaction:           t,
			FormattedAmount:       httpx.FormatPaise(t.AmountPaise),
			FormattedBalanceAfter: httpx.FormatPaise(t.BalanceAfterPaise),
		})
	}

	h.renderApp(w, r, "app_activity", "Activity Statement", "activity", map[string]any{
		"Transactions": fmtTxns,
		"FilterType":   fType,
		"FilterFrom":   fFrom,
		"FilterTo":     fTo,
	})
}

func (h *Handler) AppWithdraw(w http.ResponseWriter, r *http.Request) {
	token := httpx.ExtractToken(r)
	u, err := h.authSvc.GetUserByToken(token)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	accounts, _ := h.bankSvc.ListBankAccounts(u.ID)
	h.renderApp(w, r, "app_withdraw", "Withdraw to Bank", "banks", map[string]any{
		"BankAccounts": accounts,
	})
}

func (h *Handler) AppBanks(w http.ResponseWriter, r *http.Request) {
	token := httpx.ExtractToken(r)
	u, err := h.authSvc.GetUserByToken(token)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	accounts, _ := h.bankSvc.ListBankAccounts(u.ID)
	h.renderApp(w, r, "app_banks", "Linked Bank Accounts", "banks", map[string]any{
		"BankAccounts": accounts,
	})
}

func (h *Handler) AppProfile(w http.ResponseWriter, r *http.Request) {
	token := httpx.ExtractToken(r)
	u, err := h.authSvc.GetUserByToken(token)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	wRes, _ := h.walletSvc.GetWalletResponse(u.ID)
	sentToday := int64(0)
	dailyLimit := int64(1000000)
	pct := 0

	if wRes != nil {
		sentToday = wRes.SentTodayPaise
		dailyLimit = wRes.DailySendLimitPaise
		if dailyLimit > 0 {
			pct = int((float64(sentToday) / float64(dailyLimit)) * 100)
			if pct > 100 {
				pct = 100
			}
		}
	}

	h.renderApp(w, r, "app_profile", "Profile & Settings", "profile", map[string]any{
		"FormattedSentToday":  httpx.FormatPaise(sentToday),
		"FormattedDailyLimit": httpx.FormatPaise(dailyLimit),
		"SentTodayPercent":   pct,
	})
}

// Admin Page Handlers
func (h *Handler) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	m, _ := h.adminSvc.GetMetrics()

	type fmtTopSender struct {
		admin.TopSender
		FormattedTotal string
	}
	var fmtTop []fmtTopSender
	if m != nil {
		for _, ts := range m.TopSenders {
			fmtTop = append(fmtTop, fmtTopSender{
				TopSender:      ts,
				FormattedTotal: httpx.FormatPaise(ts.TotalPaise),
			})
		}
	}

	totalHeldStr := "₹0.00"
	todayVolStr := "₹0.00"
	if m != nil {
		totalHeldStr = httpx.FormatPaise(m.TotalBalancesHeldPaise)
		todayVolStr = httpx.FormatPaise(m.TodayVolumePaise)
	}

	h.renderAdmin(w, r, "admin_dashboard", "Admin Dashboard", "admin_dashboard", map[string]any{
		"Metrics":                m,
		"FormattedTotalBalances": totalHeldStr,
		"FormattedTodayVolume":   todayVolStr,
	})
}

func (h *Handler) AdminWallets(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	wallets, _ := h.adminSvc.ListWallets(status)

	type fmtWallet struct {
		admin.AdminWallet
		FormattedBalance string
	}
	var fmtWallets []fmtWallet
	for _, w := range wallets {
		fmtWallets = append(fmtWallets, fmtWallet{
			AdminWallet:      w,
			FormattedBalance: httpx.FormatPaise(w.BalancePaise),
		})
	}

	h.renderAdmin(w, r, "admin_wallets", "Wallets", "admin_wallets", map[string]any{
		"Wallets":      fmtWallets,
		"StatusFilter": status,
	})
}

func (h *Handler) AdminWalletDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	detail, err := h.adminSvc.GetWalletDetail(id)
	if err != nil {
		http.Redirect(w, r, "/admin/wallets", http.StatusSeeOther)
		return
	}

	type fmtTxn struct {
		wallet.Transaction
		FormattedAmount       string
		FormattedBalanceAfter string
	}
	var fmtTxns []fmtTxn
	if detail != nil {
		for _, t := range detail.Transactions {
			fmtTxns = append(fmtTxns, fmtTxn{
				Transaction:           t,
				FormattedAmount:       httpx.FormatPaise(t.AmountPaise),
				FormattedBalanceAfter: httpx.FormatPaise(t.BalanceAfterPaise),
			})
		}
	}

	h.renderAdmin(w, r, "admin_wallet_detail", "Wallet Detail", "admin_wallets", map[string]any{
		"Wallet":           detail.Wallet,
		"FormattedBalance": httpx.FormatPaise(detail.Wallet.BalancePaise),
		"Transactions":     fmtTxns,
	})
}

func (h *Handler) AdminTransfers(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	rawTransfers, _ := h.adminSvc.ListTransfers(status)

	type fmtTr struct {
		ID             string
		FromUserName   string
		FromUserHandle string
		ToUserName     string
		ToUserHandle   string
		FormattedAmount string
		Status         string
		FailureReason  *string
		CreatedAt      any
	}

	h.renderAdmin(w, r, "admin_transfers", "Transfers Log", "admin_transfers", map[string]any{
		"Transfers":    rawTransfers,
		"StatusFilter": status,
	})
}

func (h *Handler) AdminReconciliation(w http.ResponseWriter, r *http.Request) {
	check, _ := h.adminSvc.CheckLedger()

	balStr := "₹0.00"
	ledStr := "₹0.00"
	if check != nil {
		balStr = httpx.FormatPaise(check.SumOfBalances)
		ledStr = httpx.FormatPaise(check.SumOfLedger)
	}

	h.renderAdmin(w, r, "admin_reconciliation", "Reconciliation Check", "admin_reconcile", map[string]any{
		"Check":             check,
		"FormattedBalances": balStr,
		"FormattedLedger":   ledStr,
	})
}
