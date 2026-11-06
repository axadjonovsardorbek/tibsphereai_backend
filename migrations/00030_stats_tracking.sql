-- +goose Up
-- +goose StatementBegin

-- Admin statistikasi (TZ_STATISTICS): keyin tiklab bo'lmaydigan ma'lumotlarni hozirdan yozib borish.

-- Foydalanuvchi: oxirgi haqiqiy faollik (sessiya/javob/fleshkarta, login emas), ro'yxatdan o'tish
-- usuli, oxirgi so'rov kelgan platforma va ilova versiyasi.
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS signup_method VARCHAR(20) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS platform VARCHAR(20) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS app_version VARCHAR(40) NOT NULL DEFAULT '';

UPDATE users SET signup_method = CASE
    WHEN google_id IS NOT NULL THEN 'google'
    WHEN phone_number IS NOT NULL THEN 'telegram'
    WHEN email IS NOT NULL THEN 'email'
    ELSE '' END
WHERE signup_method = '';

UPDATE users u SET last_active_at = a.ts
FROM (
    SELECT user_id, MAX(ts) AS ts FROM (
        SELECT user_id, COALESCE(finished_at, started_at) AS ts FROM practice_sessions
        UNION ALL SELECT user_id, COALESCE(ended_at, started_at) FROM simulation_sessions
        UNION ALL SELECT user_id, reviewed_at FROM flashcard_reviews
    ) x GROUP BY user_id
) a
WHERE a.user_id = u.id AND u.last_active_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_last_active ON users(last_active_at);
CREATE INDEX IF NOT EXISTS idx_users_created ON users(created_at);

-- AI chaqiruvi: qaysi sessiya, model, prompt/completion tokenlar va foydalanuvchi turi
-- (premium | coin | free) chaqiruv paytidagi holat bo'yicha.
ALTER TABLE ai_usage_logs ADD COLUMN IF NOT EXISTS session_kind VARCHAR(20) NOT NULL DEFAULT '';
ALTER TABLE ai_usage_logs ADD COLUMN IF NOT EXISTS session_id UUID;
ALTER TABLE ai_usage_logs ADD COLUMN IF NOT EXISTS model VARCHAR(60) NOT NULL DEFAULT '';
ALTER TABLE ai_usage_logs ADD COLUMN IF NOT EXISTS prompt_tokens INT NOT NULL DEFAULT 0;
ALTER TABLE ai_usage_logs ADD COLUMN IF NOT EXISTS completion_tokens INT NOT NULL DEFAULT 0;
ALTER TABLE ai_usage_logs ADD COLUMN IF NOT EXISTS user_tier VARCHAR(10) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_ai_usage_logs_created ON ai_usage_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_ai_usage_logs_user ON ai_usage_logs(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_ai_usage_logs_session ON ai_usage_logs(session_id) WHERE session_id IS NOT NULL;

-- Practice: AI "tibbiyotga aloqasiz" deb rad etgan erkin mavzular.
ALTER TABLE practice_sessions ADD COLUMN IF NOT EXISTS rejected_reason VARCHAR(30) NOT NULL DEFAULT '';

-- Tanga yozuvi manbasi. Kirim: purchased | referral | promo | reward | admin.
-- Chiqim: balance (sotib olingan/yig'ilgan balansdan) | daily_free (kunlik bepul tangadan; bu qatorlar
-- balansni o'zgartirmaydi, faqat statistika uchun - admin tanga monitoringida ko'rsatilmaydi).
ALTER TABLE point_transactions ADD COLUMN IF NOT EXISTS source VARCHAR(20) NOT NULL DEFAULT '';
UPDATE point_transactions SET source = CASE
    WHEN delta < 0 THEN 'balance'
    WHEN reason = 'coin_purchase' THEN 'purchased'
    WHEN reason = 'referral_reward' THEN 'referral'
    WHEN reason = 'promo_redeem' THEN 'promo'
    ELSE 'reward' END
WHERE kind = 'coin' AND source = '';

CREATE INDEX IF NOT EXISTS idx_orders_paid_at ON orders(paid_at) WHERE status = 'PAID';
CREATE INDEX IF NOT EXISTS idx_point_transactions_created ON point_transactions(created_at);

-- 1 USD necha so'm (AI xarajatini so'mda ko'rsatish uchun).
INSERT INTO app_settings (key, value) VALUES ('usd_to_uzs_rate', '12800') ON CONFLICT (key) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM app_settings WHERE key = 'usd_to_uzs_rate';
DROP INDEX IF EXISTS idx_point_transactions_created;
DELETE FROM point_transactions WHERE source = 'daily_free';
ALTER TABLE point_transactions DROP COLUMN IF EXISTS source;
DROP INDEX IF EXISTS idx_orders_paid_at;
ALTER TABLE practice_sessions DROP COLUMN IF EXISTS rejected_reason;
DROP INDEX IF EXISTS idx_ai_usage_logs_session;
DROP INDEX IF EXISTS idx_ai_usage_logs_user;
DROP INDEX IF EXISTS idx_ai_usage_logs_created;
ALTER TABLE ai_usage_logs DROP COLUMN IF EXISTS user_tier;
ALTER TABLE ai_usage_logs DROP COLUMN IF EXISTS completion_tokens;
ALTER TABLE ai_usage_logs DROP COLUMN IF EXISTS prompt_tokens;
ALTER TABLE ai_usage_logs DROP COLUMN IF EXISTS model;
ALTER TABLE ai_usage_logs DROP COLUMN IF EXISTS session_id;
ALTER TABLE ai_usage_logs DROP COLUMN IF EXISTS session_kind;
DROP INDEX IF EXISTS idx_users_created;
DROP INDEX IF EXISTS idx_users_last_active;
ALTER TABLE users DROP COLUMN IF EXISTS app_version;
ALTER TABLE users DROP COLUMN IF EXISTS platform;
ALTER TABLE users DROP COLUMN IF EXISTS signup_method;
ALTER TABLE users DROP COLUMN IF EXISTS last_active_at;
-- +goose StatementEnd
