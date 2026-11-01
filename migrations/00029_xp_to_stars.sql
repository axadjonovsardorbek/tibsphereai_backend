-- +goose Up
-- XP -> Stars: "tajriba ballari" endi "yulduzlar" (stars). Faqat nom o'zgaradi, qiymatlar saqlanadi.
ALTER TABLE users RENAME COLUMN xp TO stars;
ALTER TABLE levels RENAME COLUMN required_xp TO required_stars;
ALTER TABLE levels RENAME COLUMN xp_reward TO stars_reward;
ALTER TABLE simulation_sessions RENAME COLUMN xp_earned TO stars_earned;
ALTER TABLE practice_sessions RENAME COLUMN xp_earned TO stars_earned;

ALTER TABLE point_transactions DROP CONSTRAINT IF EXISTS point_transactions_kind_check;
UPDATE point_transactions SET kind = 'stars' WHERE kind = 'xp';
ALTER TABLE point_transactions ADD CONSTRAINT point_transactions_kind_check CHECK (kind IN ('stars', 'coin'));

-- Reyting (davr bo'yicha yulduzlar) point_transactions'dan hisoblanadi.
CREATE INDEX IF NOT EXISTS idx_point_transactions_user_kind_created
    ON point_transactions (user_id, kind, created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_point_transactions_user_kind_created;
ALTER TABLE point_transactions DROP CONSTRAINT IF EXISTS point_transactions_kind_check;
UPDATE point_transactions SET kind = 'xp' WHERE kind = 'stars';
ALTER TABLE point_transactions ADD CONSTRAINT point_transactions_kind_check CHECK (kind IN ('xp', 'coin'));
ALTER TABLE practice_sessions RENAME COLUMN stars_earned TO xp_earned;
ALTER TABLE simulation_sessions RENAME COLUMN stars_earned TO xp_earned;
ALTER TABLE levels RENAME COLUMN stars_reward TO xp_reward;
ALTER TABLE levels RENAME COLUMN required_stars TO required_xp;
ALTER TABLE users RENAME COLUMN stars TO xp;
