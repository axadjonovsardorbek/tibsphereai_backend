-- +goose Up
-- +goose StatementBegin

-- AI real-time Case/Test (new-feature.md): admin faqat bo'lim/mavzu va manbalarni kiritadi,
-- foydalanuvchi sessiyada AI har safar yangi case/test generatsiya qiladi.

-- Mavzu uchun AI yo'nalishi (masalan "faqat pediatrik holatlar").
ALTER TABLE topics ADD COLUMN IF NOT EXISTS ai_hint TEXT NOT NULL DEFAULT '';

-- Manbalar: mavjud case_sources jadvali kengaytiriladi (eski /web/case-source ishlashda davom etadi).
-- content - manbaning ajratib olingan matni (fayl/URL uchun ham), eski qatorlarda matn yoki URL.
ALTER TABLE case_sources ADD COLUMN IF NOT EXISTS type VARCHAR(20) NOT NULL DEFAULT 'text'
    CHECK (type IN ('book', 'article', 'website', 'text'));
ALTER TABLE case_sources ADD COLUMN IF NOT EXISTS language VARCHAR(8) NOT NULL DEFAULT '';
ALTER TABLE case_sources ADD COLUMN IF NOT EXISTS author VARCHAR(200) NOT NULL DEFAULT '';
ALTER TABLE case_sources ADD COLUMN IF NOT EXISTS url TEXT NOT NULL DEFAULT '';
ALTER TABLE case_sources ADD COLUMN IF NOT EXISTS file_url TEXT NOT NULL DEFAULT '';
-- Eski qatorlar (qo'lda kiritilgan matn/havola) darhol 'ready' - ular chunklanmaydi, eski
-- usulda (to'liq matn promptga) ishlatiladi.
ALTER TABLE case_sources ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'ready'
    CHECK (status IN ('uploaded', 'processing', 'ready', 'failed'));
ALTER TABLE case_sources ADD COLUMN IF NOT EXISTS error TEXT NOT NULL DEFAULT '';
ALTER TABLE case_sources ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

-- Manbaning bo'lim/mavzularga bog'lanishi. Ikkalasi NULL bo'lgan qator bo'lmaydi;
-- manbaga umuman link yo'q bo'lsa - u umumiy manba hisoblanadi.
CREATE TABLE IF NOT EXISTS source_links (
    source_id UUID NOT NULL REFERENCES case_sources(id) ON DELETE CASCADE,
    category_id UUID REFERENCES categories(id) ON DELETE CASCADE,
    topic_id UUID REFERENCES topics(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_source_links_source ON source_links(source_id);
CREATE INDEX IF NOT EXISTS idx_source_links_category ON source_links(category_id);
CREATE INDEX IF NOT EXISTS idx_source_links_topic ON source_links(topic_id);

-- Chunklar. embedding - REAL[] (pgvector talab qilmaydi; o'xshashlik Go'da hisoblanadi),
-- embedding bo'sh bo'lsa kalit so'z bo'yicha qidiriladi.
CREATE TABLE IF NOT EXISTS source_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES case_sources(id) ON DELETE CASCADE,
    idx INT NOT NULL,
    locator VARCHAR(100) NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    tokens INT NOT NULL DEFAULT 0,
    embedding REAL[]
);
CREATE INDEX IF NOT EXISTS idx_source_chunks_source ON source_chunks(source_id);

-- Practice sessiyalar (foydalanuvchi o'zi tanlagan bo'lim/mavzu/qiyinlik/rejim).
CREATE TABLE IF NOT EXISTS practice_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    mode VARCHAR(10) NOT NULL CHECK (mode IN ('case', 'test')),
    category_id UUID NOT NULL REFERENCES categories(id),
    topic_ids UUID[] NOT NULL DEFAULT '{}',
    difficulty VARCHAR(10) NOT NULL CHECK (difficulty IN ('easy', 'medium', 'hard')),
    total_items INT NOT NULL DEFAULT 0, -- test rejimida savollar soni; case rejimida 0 = cheksiz
    status VARCHAR(12) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'finished', 'abandoned')),
    score INT NOT NULL DEFAULT 0,
    xp_earned INT NOT NULL DEFAULT 0,
    prompt_tokens INT NOT NULL DEFAULT 0,
    completion_tokens INT NOT NULL DEFAULT 0,
    language VARCHAR(4) NOT NULL DEFAULT 'uz',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_practice_sessions_user ON practice_sessions(user_id, started_at DESC);

-- Sessiyadagi har bir generatsiya qilingan case/savol. payload - AI chiqishi (to'liq,
-- to'g'ri javob bilan; userga yuborilganda yashiriladi), cases jadvaliga yozilmaydi.
CREATE TABLE IF NOT EXISTS practice_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES practice_sessions(id) ON DELETE CASCADE,
    idx INT NOT NULL,
    kind VARCHAR(10) NOT NULL CHECK (kind IN ('case', 'test')),
    payload JSONB NOT NULL,
    sources JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_chunk_ids UUID[] NOT NULL DEFAULT '{}',
    summary TEXT NOT NULL DEFAULT '',
    user_answer JSONB,
    evaluation JSONB,
    is_correct BOOLEAN,
    score INT,
    reported BOOLEAN NOT NULL DEFAULT FALSE,
    report_reason TEXT NOT NULL DEFAULT '',
    prompt_version VARCHAR(40) NOT NULL DEFAULT '',
    model VARCHAR(60) NOT NULL DEFAULT '',
    tokens_used INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    answered_at TIMESTAMP WITH TIME ZONE,
    UNIQUE (session_id, idx)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS practice_items;
DROP TABLE IF EXISTS practice_sessions;
DROP TABLE IF EXISTS source_chunks;
DROP TABLE IF EXISTS source_links;
ALTER TABLE case_sources DROP COLUMN IF EXISTS is_active;
ALTER TABLE case_sources DROP COLUMN IF EXISTS error;
ALTER TABLE case_sources DROP COLUMN IF EXISTS status;
ALTER TABLE case_sources DROP COLUMN IF EXISTS file_url;
ALTER TABLE case_sources DROP COLUMN IF EXISTS url;
ALTER TABLE case_sources DROP COLUMN IF EXISTS author;
ALTER TABLE case_sources DROP COLUMN IF EXISTS language;
ALTER TABLE case_sources DROP COLUMN IF EXISTS type;
ALTER TABLE topics DROP COLUMN IF EXISTS ai_hint;

-- +goose StatementEnd
