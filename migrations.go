package main

import (
	"database/sql"
	"log"
)

// RunMigrations executes database migrations
func RunMigrations(db *sql.DB) error {
	log.Println("Running database migrations...")

	migrations := []struct {
		version int
		name    string
		sql     string
	}{
		{
			version: 1,
			name:    "create_migration_table",
			sql: `
				CREATE TABLE IF NOT EXISTS schema_migrations (
					version INTEGER PRIMARY KEY,
					name TEXT NOT NULL,
					applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);
			`,
		},
		{
			version: 2,
			name:    "add_user_columns_to_transactions",
			sql: `
				-- Check if user_id column exists, if not add it
				CREATE TABLE IF NOT EXISTS transactions_new (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					user_id INTEGER,
					type TEXT NOT NULL,
					amount INTEGER NOT NULL,
					category_id INTEGER,
					note TEXT,
					date DATETIME DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY(user_id) REFERENCES users(id),
					FOREIGN KEY(category_id) REFERENCES categories(id)
				);
				
				-- Copy existing data
				INSERT OR IGNORE INTO transactions_new (id, type, amount, category_id, date)
				SELECT id, type, amount, category_id, date FROM transactions;
				
				-- Drop old table and rename
				DROP TABLE IF EXISTS transactions_old;
				ALTER TABLE transactions RENAME TO transactions_old;
				ALTER TABLE transactions_new RENAME TO transactions;
				DROP TABLE IF EXISTS transactions_old;
			`,
		},
		{
			version: 3,
			name:    "update_budgets_structure",
			sql: `
				CREATE TABLE IF NOT EXISTS budgets_new (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					user_id INTEGER,
					category_id INTEGER NOT NULL,
					amount_limit INTEGER NOT NULL,
					month_year TEXT NOT NULL DEFAULT (strftime('%Y-%m', 'now')),
					UNIQUE(user_id, category_id, month_year),
					FOREIGN KEY(user_id) REFERENCES users(id),
					FOREIGN KEY(category_id) REFERENCES categories(id)
				);
				
				-- Copy existing data
				INSERT OR IGNORE INTO budgets_new (id, category_id, amount_limit, month_year)
				SELECT id, category_id, amount_limit, strftime('%Y-%m', 'now')
				FROM budgets;
				
				-- Drop old table and rename
				DROP TABLE IF EXISTS budgets_old;
				ALTER TABLE budgets RENAME TO budgets_old;
				ALTER TABLE budgets_new RENAME TO budgets;
				DROP TABLE IF EXISTS budgets_old;
			`,
		},
		{
			version: 4,
			name:    "create_savings_tables",
			sql: `
				CREATE TABLE IF NOT EXISTS savings (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					user_id INTEGER NOT NULL,
					name TEXT NOT NULL,
					type TEXT NOT NULL DEFAULT 'savings',
					icon TEXT NOT NULL DEFAULT 'ph-piggy-bank',
					color TEXT NOT NULL DEFAULT '#c3f545',
					target_amount INTEGER NOT NULL DEFAULT 0,
					current_amount INTEGER NOT NULL DEFAULT 0,
					deadline TEXT,
					description TEXT,
					is_completed INTEGER NOT NULL DEFAULT 0,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY(user_id) REFERENCES users(id)
				);

				CREATE TABLE IF NOT EXISTS savings_transactions (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					savings_id INTEGER NOT NULL,
					user_id INTEGER NOT NULL,
					type TEXT NOT NULL,
					amount INTEGER NOT NULL,
					note TEXT,
					date DATETIME DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY(savings_id) REFERENCES savings(id),
					FOREIGN KEY(user_id) REFERENCES users(id)
				);
			`,
		},
		{
			version: 5,
			name:    "savings_link_to_transactions",
			sql: `
				ALTER TABLE savings_transactions ADD COLUMN linked_transaction_id INTEGER;

				INSERT OR IGNORE INTO categories (name, type, icon)
				VALUES
					('Transfer ke Tabungan', 'expense', 'ph-piggy-bank'),
					('Tarik dari Tabungan', 'income', 'ph-piggy-bank');
			`,
		},
		{
			version: 6,
			name:    "add_receipt_to_transactions",
			sql: `
				ALTER TABLE transactions ADD COLUMN receipt_path TEXT;
			`,
		},
		{
			version: 7,
			name:    "add_avatar_to_users",
			sql: `
				ALTER TABLE users ADD COLUMN avatar TEXT;
			`,
		},
	}

	// Get current migration version
	var currentVersion int
	db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&currentVersion)

	// Run pending migrations
	for _, migration := range migrations {
		if migration.version <= currentVersion {
			continue
		}

		log.Printf("Applying migration %d: %s", migration.version, migration.name)

		// Execute migration
		_, err := db.Exec(migration.sql)
		if err != nil {
			log.Printf("Migration %d failed: %v", migration.version, err)
			return err
		}

		// Record migration
		_, err = db.Exec(`
			INSERT INTO schema_migrations (version, name)
			VALUES (?, ?)
		`, migration.version, migration.name)

		if err != nil {
			log.Printf("Failed to record migration %d: %v", migration.version, err)
			return err
		}

		log.Printf("Migration %d completed successfully", migration.version)
	}

	log.Println("All migrations completed")
	return nil
}

// CreateDefaultAdmin creates a default admin user if none exists
func CreateDefaultAdmin() error {
	// Check if any users exist
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		log.Println("Users already exist, skipping default admin creation")
		return nil
	}

	log.Println("Creating default admin user...")

	// Create default admin
	authService := NewAuthService()
	passwordHash, err := authService.HashPassword("admin123")
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO users (name, email, password_hash)
		VALUES (?, ?, ?)
	`, "Administrator", "admin@fintrack.id", passwordHash)

	if err != nil {
		return err
	}

	log.Println("Default admin created successfully")
	log.Println("Email: admin@fintrack.id")
	log.Println("Password: admin123")
	log.Println("⚠️  IMPORTANT: Change this password in production!")

	return nil
}
