package db

const Schema = `
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    phone TEXT NOT NULL,
    handle TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('user', 'admin')),
    kyc_level TEXT NOT NULL CHECK(kyc_level IN ('basic', 'full')),
    created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS wallets (
    id TEXT PRIMARY KEY,
    user_id TEXT UNIQUE NOT NULL REFERENCES users(id),
    balance_paise INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL CHECK(status IN ('active', 'frozen')),
    daily_send_limit_paise INTEGER NOT NULL,
    created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    wallet_id TEXT NOT NULL REFERENCES wallets(id),
    type TEXT NOT NULL CHECK(type IN ('top_up', 'transfer_out', 'transfer_in', 'withdrawal', 'fee', 'refund')),
    amount_paise INTEGER NOT NULL,
    balance_after_paise INTEGER NOT NULL,
    reference_id TEXT NOT NULL,
    counterparty_wallet_id TEXT REFERENCES wallets(id),
    note TEXT NOT NULL,
    created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS transfers (
    id TEXT PRIMARY KEY,
    from_wallet_id TEXT NOT NULL REFERENCES wallets(id),
    to_wallet_id TEXT NOT NULL REFERENCES wallets(id),
    amount_paise INTEGER NOT NULL,
    note TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('completed', 'failed')),
    failure_reason TEXT,
    created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS money_requests (
    id TEXT PRIMARY KEY,
    from_user_id TEXT NOT NULL REFERENCES users(id),
    to_user_id TEXT NOT NULL REFERENCES users(id),
    amount_paise INTEGER NOT NULL,
    note TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('pending', 'paid', 'declined', 'cancelled')),
    created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS bank_accounts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    bank_name TEXT NOT NULL,
    account_holder TEXT NOT NULL,
    account_last4 TEXT NOT NULL,
    ifsc TEXT NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS withdrawals (
    id TEXT PRIMARY KEY,
    wallet_id TEXT NOT NULL REFERENCES wallets(id),
    bank_account_id TEXT NOT NULL REFERENCES bank_accounts(id),
    amount_paise INTEGER NOT NULL,
    fee_paise INTEGER NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('processing', 'completed')),
    created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    token TEXT UNIQUE NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(id),
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_handle ON users(handle);
CREATE INDEX IF NOT EXISTS idx_wallets_user_id ON wallets(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_wallet_created ON transactions(wallet_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_reference ON transactions(reference_id);
CREATE INDEX IF NOT EXISTS idx_transfers_from_wallet ON transfers(from_wallet_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transfers_to_wallet ON transfers(to_wallet_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_money_requests_from ON money_requests(from_user_id);
CREATE INDEX IF NOT EXISTS idx_money_requests_to ON money_requests(to_user_id);
CREATE INDEX IF NOT EXISTS idx_bank_accounts_user ON bank_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_withdrawals_wallet ON withdrawals(wallet_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
`
