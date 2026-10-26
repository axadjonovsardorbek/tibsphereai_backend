-- +goose Up
-- +goose StatementBegin

-- AI yo'l xaritasi keshi (TZ_MVP): bilim xaritasi har so'rovda hisoblanadi (arzon), AI'ning yozma
-- tavsiyalari esa tokenga pul yechgani uchun saqlanadi va foydalanuvchi so'raganda yangilanadi.
CREATE TABLE IF NOT EXISTS user_roadmaps (
    user_id UUID PRIMARY KEY REFERENCES users(id),
    payload JSONB NOT NULL,
    language VARCHAR(4) NOT NULL DEFAULT 'uz',
    generated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

-- Foydalanuvchilar fikri (TZ_MVP): ilova ichidagi baho/izoh.
CREATE TABLE IF NOT EXISTS user_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    rating INT CHECK (rating BETWEEN 1 AND 5),
    kind VARCHAR(20) NOT NULL DEFAULT 'general' CHECK (kind IN ('general', 'bug', 'idea', 'content', 'nps')),
    message TEXT NOT NULL DEFAULT '',
    context VARCHAR(100) NOT NULL DEFAULT '',   -- masalan: practice_session:<id>, flashcards, roadmap
    app_version VARCHAR(40) NOT NULL DEFAULT '',
    status VARCHAR(10) NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'seen', 'done')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_user_feedback_created ON user_feedback(created_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS user_feedback;
DROP TABLE IF EXISTS user_roadmaps;

-- +goose StatementEnd
