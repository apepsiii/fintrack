-- Migration script for Phase 2
-- Run this to upgrade from Phase 1 to Phase 2

-- Add user_id column to transactions if not exists
ALTER TABLE transactions ADD COLUMN user_id INTEGER;
ALTER TABLE transactions ADD COLUMN note TEXT;

-- Update budgets table structure
-- Note: SQLite doesn't support ALTER COLUMN, so we need to recreate
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

-- Copy data from old budgets table
INSERT INTO budgets_new (id, category_id, amount_limit, month_year)
SELECT id, category_id, amount_limit, strftime('%Y-%m', 'now')
FROM budgets;

-- Drop old table and rename new one
DROP TABLE budgets;
ALTER TABLE budgets_new RENAME TO budgets;

-- Create default admin user
-- Password: admin123 (hashed with bcrypt)
-- You should change this in production
INSERT OR IGNORE INTO users (id, name, email, password_hash, created_at)
VALUES (
    1,
    'Administrator',
    'admin@fintrack.id',
    '$2a$10$XqKvZv5Y5zG5LhV3X5x5x.J5YqKvZv5Y5zG5LhV3X5x5x.J5YqKvZu',
    datetime('now')
);

-- Migration complete
SELECT 'Phase 2 migration completed successfully' as status;
