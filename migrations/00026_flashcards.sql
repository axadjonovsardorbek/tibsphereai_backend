-- +goose Up
-- +goose StatementBegin

-- TZ_MVP: AI fleshkartalar (2 tomonlama, intervalli takrorlash). Kartalar xatolar asosida
-- (sessiya yakunida) yoki mavzu bo'yicha (deck buyurtmasi) yaratiladi; kunlik takrorlash bepul.

CREATE TABLE IF NOT EXISTS flashcard_decks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    title VARCHAR(200) NOT NULL,
    kind VARCHAR(10) NOT NULL DEFAULT 'topic' CHECK (kind IN ('mistakes', 'topic', 'manual')),
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    topic_id UUID REFERENCES topics(id) ON DELETE SET NULL,
    free_topic VARCHAR(200) NOT NULL DEFAULT '',
    language VARCHAR(4) NOT NULL DEFAULT 'uz',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at BIGINT DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_flashcard_decks_user ON flashcard_decks(user_id) WHERE deleted_at = 0;

CREATE TABLE IF NOT EXISTS flashcards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    deck_id UUID REFERENCES flashcard_decks(id) ON DELETE SET NULL,
    front TEXT NOT NULL,
    back TEXT NOT NULL,
    source_refs JSONB NOT NULL DEFAULT '[]'::jsonb,   -- [{source_id,type,title,author,url,locator}]
    origin_session_id UUID REFERENCES practice_sessions(id) ON DELETE SET NULL,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    topic_id UUID REFERENCES topics(id) ON DELETE SET NULL,
    language VARCHAR(4) NOT NULL DEFAULT 'uz',
    -- Intervalli takrorlash holati (SM-2 o'xshash, TZ: 1 -> 3 -> 7 -> 21 kun)
    ease REAL NOT NULL DEFAULT 2.5,
    interval_days INT NOT NULL DEFAULT 0,
    repetitions INT NOT NULL DEFAULT 0,
    lapses INT NOT NULL DEFAULT 0,
    due_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    last_reviewed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(10) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at BIGINT DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_flashcards_due ON flashcards(user_id, due_at) WHERE deleted_at = 0 AND status = 'active';
CREATE INDEX IF NOT EXISTS idx_flashcards_deck ON flashcards(deck_id) WHERE deleted_at = 0;
CREATE INDEX IF NOT EXISTS idx_flashcards_topic ON flashcards(topic_id) WHERE deleted_at = 0;

CREATE TABLE IF NOT EXISTS flashcard_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id UUID NOT NULL REFERENCES flashcards(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    rating VARCHAR(5) NOT NULL CHECK (rating IN ('again', 'hard', 'good', 'easy')),
    interval_before INT NOT NULL DEFAULT 0,
    interval_after INT NOT NULL DEFAULT 0,
    reviewed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_flashcard_reviews_user_day ON flashcard_reviews(user_id, reviewed_at);
CREATE INDEX IF NOT EXISTS idx_flashcard_reviews_card ON flashcard_reviews(card_id);

-- Practice elementlari mavzu bo'yicha tahlil qilinishi uchun (fleshkarta monitoringi, yo'l xaritasi):
-- element qaysi mavzudan ekani saqlanadi (aralash sessiyada AI nomini aytadi, backend id'ga moslaydi).
ALTER TABLE practice_items ADD COLUMN IF NOT EXISTS topic_id UUID REFERENCES topics(id) ON DELETE SET NULL;
ALTER TABLE practice_items ADD COLUMN IF NOT EXISTS category_id UUID REFERENCES categories(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_practice_items_topic ON practice_items(topic_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE practice_items DROP COLUMN IF EXISTS category_id;
ALTER TABLE practice_items DROP COLUMN IF EXISTS topic_id;
DROP TABLE IF EXISTS flashcard_reviews;
DROP TABLE IF EXISTS flashcards;
DROP TABLE IF EXISTS flashcard_decks;

-- +goose StatementEnd
