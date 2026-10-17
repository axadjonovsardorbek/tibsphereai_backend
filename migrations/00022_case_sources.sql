-- +goose Up
-- +goose StatementBegin

-- AI case generatsiyasi uchun ishonchli manbalar (veb-havola yoki matn parchasi).
-- Admin bir marta kiritadi; generatsiyada tanlanmasa hammasi ishlatiladi.
CREATE TABLE IF NOT EXISTS case_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at BIGINT DEFAULT 0
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS case_sources;

-- +goose StatementEnd
