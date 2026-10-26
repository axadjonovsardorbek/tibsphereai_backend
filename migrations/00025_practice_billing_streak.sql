-- +goose Up
-- +goose StatementBegin

-- TZ_MVP: Pay-as-you-go. Practice sessiyalari AI token sarfiga qarab tanga yechadi;
-- har kuni har foydalanuvchiga bepul tangalar beriladi (eskirib ketadi, to'planmaydi).
ALTER TABLE practice_sessions ADD COLUMN IF NOT EXISTS coin_billed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE practice_sessions ADD COLUMN IF NOT EXISTS coins_charged INT NOT NULL DEFAULT 0;
-- Erkin mavzu: tayyor bo'lim/mavzusiz, foydalanuvchi o'zi yozgan mavzu.
ALTER TABLE practice_sessions ALTER COLUMN category_id DROP NOT NULL;
ALTER TABLE practice_sessions ADD COLUMN IF NOT EXISTS free_topic VARCHAR(200) NOT NULL DEFAULT '';
-- Sessiya yakunida fonda yaratiladigan fleshkartalar (xatolar asosida).
ALTER TABLE practice_sessions ADD COLUMN IF NOT EXISTS flashcards_status VARCHAR(10) NOT NULL DEFAULT 'none'
    CHECK (flashcards_status IN ('none', 'pending', 'done', 'failed'));
ALTER TABLE practice_sessions ADD COLUMN IF NOT EXISTS flashcards_created INT NOT NULL DEFAULT 0;

-- Kunlik bepul tangalar: kun (Asia/Tashkent) bo'yicha bitta qator; granted - o'sha kuni berilgan,
-- used - sarflangani. Balansga qo'shilmaydi: yangi kunda yangidan beriladi.
CREATE TABLE IF NOT EXISTS user_daily_coins (
    user_id UUID NOT NULL REFERENCES users(id),
    day DATE NOT NULL,
    granted INT NOT NULL DEFAULT 0,
    used INT NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, day)
);
INSERT INTO app_settings (key, value) VALUES ('daily_free_coins', '10') ON CONFLICT (key) DO NOTHING;

-- Streak: ketma-ket faol kunlar (users.streak_count mavjud edi, lekin yangilanmas edi).
ALTER TABLE users ADD COLUMN IF NOT EXISTS streak_last_day DATE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS longest_streak INT NOT NULL DEFAULT 0;

-- Bo'lim/mavzu: faol/nofaol va tavsif (new-feature 4.1.1).
ALTER TABLE categories ADD COLUMN IF NOT EXISTS description JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE categories ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE topics ADD COLUMN IF NOT EXISTS description JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE topics ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE topics DROP COLUMN IF EXISTS is_active;
ALTER TABLE topics DROP COLUMN IF EXISTS description;
ALTER TABLE categories DROP COLUMN IF EXISTS is_active;
ALTER TABLE categories DROP COLUMN IF EXISTS description;
ALTER TABLE users DROP COLUMN IF EXISTS longest_streak;
ALTER TABLE users DROP COLUMN IF EXISTS streak_last_day;
DELETE FROM app_settings WHERE key = 'daily_free_coins';
DROP TABLE IF EXISTS user_daily_coins;
ALTER TABLE practice_sessions DROP COLUMN IF EXISTS flashcards_created;
ALTER TABLE practice_sessions DROP COLUMN IF EXISTS flashcards_status;
ALTER TABLE practice_sessions DROP COLUMN IF EXISTS free_topic;
ALTER TABLE practice_sessions DROP COLUMN IF EXISTS coins_charged;
ALTER TABLE practice_sessions DROP COLUMN IF EXISTS coin_billed;

-- +goose StatementEnd
