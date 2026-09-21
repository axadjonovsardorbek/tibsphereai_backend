-- +goose Up
-- +goose StatementBegin
SET TIME ZONE 'Asia/Tashkent';

-- ============ Auth / Admin ============

-- roles - admin panel rollari (superadmin + qo'shimcha rollar). superadmin - tizim roli
-- (is_system=true), o'chirilmaydi. Casbin siyosati (api/casbin/policy.csv) bilan bog'lanish
-- va role_id bo'yicha huquqlarni cheklash - keyingi bosqichda (kod hozircha yozilmagan).
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) UNIQUE NOT NULL,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at BIGINT DEFAULT 0
);

INSERT INTO roles (name, slug, is_system) VALUES ('Super Admin', 'superadmin', true) ON CONFLICT (slug) DO NOTHING;

-- custom_permissions - admin panel modullari (U-Code custom-permission patterni bilan bir xil:
-- API_DOCUMENTATION_EN.md). Daraxt: parent_id NULL bo'lsa "root modul" (masalan "Bo'limlar"),
-- parent_id bilan bo'lsa shu modulning bo'lim-modullari. Hozircha faqat boshqaruv (CRUD) -
-- haqiqiy ruxsat tekshiruvi hamon Casbin policy.csv orqali (keyingi bosqichda bog'lanadi).
CREATE TABLE IF NOT EXISTS custom_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(150) NOT NULL,
    parent_id UUID REFERENCES custom_permissions(id) DEFAULT NULL,
    attributes JSONB DEFAULT '{}'::jsonb,
    order_num BIGINT DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at BIGINT DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_custom_permissions_parent ON custom_permissions(parent_id) WHERE deleted_at = 0;

-- role_accesses - bitta rol uchun bitta modulga berilgan ruxsatlar (o'qish/yozish/yangilash/
-- o'chirish). (role_id, custom_permission_id) bo'yicha bitta yozuv - qayta belgilashda upsert.
CREATE TABLE IF NOT EXISTS role_accesses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL REFERENCES roles(id),
    custom_permission_id UUID NOT NULL REFERENCES custom_permissions(id),
    can_read BOOLEAN NOT NULL DEFAULT FALSE,
    can_write BOOLEAN NOT NULL DEFAULT FALSE,
    can_update BOOLEAN NOT NULL DEFAULT FALSE,
    can_delete BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (role_id, custom_permission_id)
);

CREATE TABLE IF NOT EXISTS admins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    role_id UUID REFERENCES roles(id) DEFAULT NULL,
    -- Faqat "Hamkor Admin" (partner_admin) roli uchun to'ldiriladi - shu hamkorning promokodlarigagina
    -- ruxsati bo'ladi. FK constraint pastda qo'shiladi (partners jadvali keyinroq yaratiladi).
    partner_id UUID DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO admins (login, password, role_id)
SELECT 'medicai', '$2a$10$DCHM3DqLWoA.lgdqM7Tkk.Qdq/OHMkBq5DaM6TCYpQQKmdF7tmfQW', id
FROM roles WHERE slug = 'superadmin'
ON CONFLICT (login) DO NOTHING;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) DEFAULT NULL,
    phone_number VARCHAR(20) DEFAULT NULL,
    email VARCHAR(150) DEFAULT NULL,
    google_id VARCHAR(100) DEFAULT NULL,
    image_url VARCHAR(255) DEFAULT NULL,
    language VARCHAR(12) DEFAULT 'uz' CHECK (language IN ('uz', 'ru', 'en')),
    specialization VARCHAR(20) DEFAULT NULL CHECK (specialization IS NULL OR specialization IN ('student', 'resident', 'doctor')),
    level INT NOT NULL DEFAULT 1,
    xp INT NOT NULL DEFAULT 0,
    coins INT NOT NULL DEFAULT 0,
    streak_count INT NOT NULL DEFAULT 0,
    last_active_day DATE DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at BIGINT DEFAULT 0,
    UNIQUE (phone_number, deleted_at),
    UNIQUE (email, deleted_at),
    UNIQUE (google_id, deleted_at)
);

-- user_devices - ko'p qurilma push support (bitta user bir nechta qurilmadan kirsa hammasiga
-- push boradi). users.fcm_token o'rnini bosadi. Storage/service kodi hozircha yozilmagan.
CREATE TABLE IF NOT EXISTS user_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    fcm_token VARCHAR(255) NOT NULL,
    platform VARCHAR(20) DEFAULT NULL CHECK (platform IS NULL OR platform IN ('ios', 'android', 'web')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (fcm_token)
);

CREATE INDEX IF NOT EXISTS idx_user_devices_user_id ON user_devices(user_id);

CREATE TABLE IF NOT EXISTS user_activities (
    user_id UUID REFERENCES users(id),
    activity BIGINT DEFAULT 0,
    day DATE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, day)
);

-- Kunlik bepul urinishlar limiti (BACKEND_STRUCTURE.md 6-bo'lim)
CREATE TABLE IF NOT EXISTS user_daily_limits (
    user_id UUID REFERENCES users(id),
    day DATE NOT NULL,
    free_attempts_used INT NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, day)
);

-- ============ Home / Kategoriyalar ============

CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name JSONB NOT NULL,
    icon_url VARCHAR(255) DEFAULT NULL,
    order_num BIGINT DEFAULT 1,
    -- coins_per_case - shu bo'limdagi bitta case'ni ochish narxi (tangada). Admin real
    -- raxodni hisoblab qo'yadi (masalan Premium 50 case = 50 tanga bo'lsa, shu yerda 1).
    coins_per_case INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at BIGINT DEFAULT 0
);

-- topics - Bo'lim (categories) ichidagi Mavzular (TZ 6-bo'lim: Bo'limlar -> Mavzular -> Case'lar).
CREATE TABLE IF NOT EXISTS topics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID NOT NULL REFERENCES categories(id),
    name JSONB NOT NULL,
    order_num BIGINT DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at BIGINT DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_topics_category ON topics(category_id) WHERE deleted_at = 0;

-- category_unlocks - foydalanuvchi tanga to'lab bitta bo'limdan N ta case ishlash
-- huquqini sotib oladi (Premium obunasiz, TZ: "50 ta case = 50 tanga" pattern).
-- Premium obuna faol bo'lsa yoki kunlik bepul limit bor bo'lsa bu jadval tekshirilmaydi -
-- faqat ikkalasi ham tugagan holatda ishlatiladi (SimulationService.Start).
CREATE TABLE IF NOT EXISTS category_unlocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    category_id UUID NOT NULL REFERENCES categories(id),
    cases_remaining INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, category_id)
);

CREATE TABLE IF NOT EXISTS study_plans (
    user_id UUID PRIMARY KEY REFERENCES users(id),
    remind_time TIME NOT NULL DEFAULT '20:00',
    remind_minutes_before INT NOT NULL DEFAULT 15,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS banners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title JSONB NOT NULL,
    description JSONB DEFAULT NULL,
    image_url JSONB NOT NULL,
    link_url VARCHAR(255) DEFAULT NULL,
    order_num BIGINT DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at BIGINT DEFAULT 0
);

-- ============ Klinik keyslar & Sevimlilar ============

CREATE TABLE IF NOT EXISTS cases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    topic_id UUID REFERENCES topics(id),
    title JSONB NOT NULL,
    subtitle JSONB DEFAULT NULL,
    -- Admin kiritadigan taxminiy javob (AI debrief/scoring uchun asos, TZ 6-bo'lim).
    expected_answer TEXT DEFAULT NULL,
    cover_image_url VARCHAR(255) DEFAULT NULL,
    difficulty VARCHAR(20) NOT NULL CHECK (difficulty IN ('easy', 'medium', 'hard')),
    chief_complaint JSONB DEFAULT NULL,
    patient_age INT DEFAULT NULL,
    patient_gender VARCHAR(10) DEFAULT NULL,
    expected_duration_minutes INT DEFAULT 10,
    initial_vitals JSONB DEFAULT '{}'::jsonb,
    visual_state VARCHAR(50) DEFAULT NULL,
    scenario JSONB DEFAULT '{}'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    is_ai_generated BOOLEAN NOT NULL DEFAULT FALSE,
    created_by UUID REFERENCES admins(id) DEFAULT NULL,
    order_num BIGINT DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at BIGINT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS favorites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    case_id UUID NOT NULL REFERENCES cases(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, case_id)
);

-- ============ Simulyatsiya dvigateli ============

CREATE TABLE IF NOT EXISTS simulation_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    case_id UUID NOT NULL REFERENCES cases(id),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'failed', 'abandoned')),
    health_percent INT NOT NULL DEFAULT 100,
    time_limit_seconds INT DEFAULT 600,
    final_score INT DEFAULT NULL,
    xp_earned INT DEFAULT 0,
    coins_earned INT DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE TABLE IF NOT EXISTS simulation_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES simulation_sessions(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('question', 'exam', 'lab', 'imaging', 'medication', 'procedure', 'finish')),
    payload JSONB DEFAULT '{}'::jsonb,
    is_correct BOOLEAN DEFAULT NULL,
    health_delta INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============ Debriefing ============

CREATE TABLE IF NOT EXISTS debrief_reports (
    session_id UUID PRIMARY KEY REFERENCES simulation_sessions(id) ON DELETE CASCADE,
    correct_steps JSONB DEFAULT '[]'::jsonb,
    incorrect_steps JSONB DEFAULT '[]'::jsonb,
    guideline_notes TEXT DEFAULT NULL,
    weak_topics JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============ Gamifikatsiya ============

-- Statik level jadvali (dinamik formula emas - admin panel orqali CRUD)
CREATE TABLE IF NOT EXISTS levels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    level_number INT UNIQUE NOT NULL,
    slug VARCHAR(50) UNIQUE NOT NULL,
    required_xp INT NOT NULL,
    title VARCHAR(100) DEFAULT NULL,
    badge_image_url VARCHAR(255) DEFAULT NULL,
    order_num BIGINT DEFAULT 1
);

INSERT INTO levels (level_number, slug, required_xp, title) VALUES
    (1, 'beginner', 0, 'Boshlang''ich'),
    (2, 'student', 500, 'Talaba'),
    (3, 'practitioner', 1500, 'Amaliyotchi'),
    (4, 'resident', 3500, 'Rezident'),
    (5, 'specialist', 7000, 'Mutaxassis')
ON CONFLICT (level_number) DO NOTHING;

CREATE TABLE IF NOT EXISTS point_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    kind VARCHAR(10) NOT NULL CHECK (kind IN ('xp', 'coin')),
    delta INT NOT NULL,
    reason VARCHAR(100) DEFAULT NULL,
    reference_type VARCHAR(50) DEFAULT NULL,
    reference_id VARCHAR(100) DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS app_settings (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO app_settings (key, value) VALUES ('daily_free_limit', '3') ON CONFLICT (key) DO NOTHING;
-- 1 tanga = necha so'm (obunani aralash/tanga bilan to'lash uchun, TZ 2.3/6-bo'lim).
INSERT INTO app_settings (key, value) VALUES ('coin_to_uzs_rate', '100') ON CONFLICT (key) DO NOTHING;

-- ============ Obuna, Hamkorlar, Promokod, To'lov ============

CREATE TABLE IF NOT EXISTS tariffs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(64) NOT NULL,
    duration BIGINT NOT NULL,
    price NUMERIC(12,2) NOT NULL DEFAULT 0,
    description VARCHAR(128) DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at BIGINT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    tariff_id UUID NOT NULL REFERENCES tariffs(id),
    -- orders.id - Payme refund (state 2->-2) bo'lganda shu bog'lanish orqali topib "refunded" qilinadi.
    -- FK constraint pastda qo'shiladi (orders jadvali shu fayl ichida keyinroq yaratiladi).
    order_id UUID DEFAULT NULL,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'expired', 'refunded')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_order_id ON subscriptions(order_id) WHERE order_id IS NOT NULL;

-- partners - Hamkor o'quv markazlari (B2B, BACKEND_STRUCTURE.md/TZ 5-6-bo'lim). Har biri mobil
-- ilovadagi tavsiya banneri sifatida ko'rsatiladi VA o'z promo_codes'larini yaratadi (Hamkor Admin
-- roli orqali - admins.partner_id shu yerga bog'lanadi).
CREATE TABLE IF NOT EXISTS partners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(150) NOT NULL,
    image_url VARCHAR(255) DEFAULT NULL,
    link_url VARCHAR(255) DEFAULT NULL,
    description JSONB DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at BIGINT DEFAULT 0
);

ALTER TABLE admins ADD CONSTRAINT fk_admins_partner_id
    FOREIGN KEY (partner_id) REFERENCES partners(id);

-- promo_codes - Hamkor (yoki Super Admin) tomonidan OLDINDAN yaratiladi, tanga qiymati bilan
-- (TZ 5-bo'lim: to'lovga bog'liq avtomatik chiqarilmaydi). Talaba kodni kiritganda used_by/used_at
-- to'ldiriladi va tizim avtomatik status='used' qiladi (admin qo'lda o'zgartirmaydi).
CREATE TABLE IF NOT EXISTS promo_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(32) UNIQUE NOT NULL,
    partner_id UUID NOT NULL REFERENCES partners(id),
    value INT NOT NULL DEFAULT 0,
    created_by UUID REFERENCES admins(id) DEFAULT NULL,
    used_by UUID REFERENCES users(id) DEFAULT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'used', 'expired')),
    issued_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    used_at TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

-- Orders + Click to'lov (imed'dagi order/order_click patterni, course_id o'rniga tariff_id)
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    tariff_id UUID NOT NULL REFERENCES tariffs(id),
    duration INT NOT NULL,
    amount NUMERIC(12,2) NOT NULL,
    tariff_amount NUMERIC(12,2) NOT NULL,
    -- Aralash to'lov: tangalar app_settings.coin_to_uzs_rate bo'yicha so'mga aylantirilib
    -- amount'dan kamaytiriladi. amount<=0 bo'lsa gateway chaqirilmasdan to'g'ridan-to'g'ri PAID.
    coins_used INT NOT NULL DEFAULT 0,
    payment_type VARCHAR(20) NOT NULL DEFAULT 'click' CHECK (payment_type IN ('click', 'payme', 'coins', 'admin')),
    status VARCHAR(20) NOT NULL DEFAULT 'NEW' CHECK (status IN ('NEW', 'RESERVED', 'PAID', 'CANCELLED', 'EXPIRED')),
    click_trans_id BIGINT DEFAULT NULL,
    click_paydoc_id BIGINT DEFAULT NULL,
    merchant_prepare_id BIGINT DEFAULT NULL,
    -- Payme JSON-RPC (docs: PaymeRequest.params.id) - state machine 1/2/-1/-2
    payme_id VARCHAR(64) UNIQUE DEFAULT NULL,
    payme_state INT DEFAULT NULL,
    payme_create_time BIGINT DEFAULT NULL,
    payme_perform_time BIGINT DEFAULT NULL,
    payme_cancel_time BIGINT DEFAULT NULL,
    payme_reason INT DEFAULT NULL,
    meta JSONB DEFAULT '{}'::jsonb,
    paid_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE subscriptions ADD CONSTRAINT fk_subscriptions_order_id
    FOREIGN KEY (order_id) REFERENCES orders(id);

-- ============ Admin/AI ============

CREATE TABLE IF NOT EXISTS ai_prompts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key VARCHAR(50) UNIQUE NOT NULL CHECK (key IN ('patient_persona', 'debrief', 'case_generation')),
    template TEXT NOT NULL,
    model_params JSONB DEFAULT '{}'::jsonb,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ai_usage_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feature VARCHAR(30) NOT NULL,
    tokens_used INT DEFAULT 0,
    cost_usd NUMERIC(12,4) DEFAULT 0,
    user_id UUID REFERENCES users(id) DEFAULT NULL,
    case_id UUID REFERENCES cases(id) DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS master_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title JSONB NOT NULL,
    message JSONB NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('selected', 'all')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    master_notification_id UUID NOT NULL REFERENCES master_notifications(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_read BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(master_notification_id, user_id)
);

-- ============ Umumiy sahifalar (about/faq/contact/app-route) ============

CREATE TABLE IF NOT EXISTS faqs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question JSONB NOT NULL,
    answer JSONB NOT NULL,
    order_num BIGINT DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at BIGINT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS abouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title JSONB NOT NULL,
    description JSONB NOT NULL,
    link_url VARCHAR(255) DEFAULT NULL,
    order_num BIGINT DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at BIGINT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    phone_number VARCHAR(100) NOT NULL,
    link_url VARCHAR(255) DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at BIGINT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS app_routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    call_center VARCHAR(20) NOT NULL,
    support_url VARCHAR(255) NOT NULL,
    app_version JSONB NOT NULL DEFAULT '{}'::jsonb,
    app_links JSONB NOT NULL DEFAULT '{}'::jsonb,
    payment_min_version VARCHAR(20) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============ Indekslar ============

CREATE INDEX IF NOT EXISTS idx_cases_topic ON cases(topic_id) WHERE deleted_at = 0;
CREATE INDEX IF NOT EXISTS idx_cases_status ON cases(status) WHERE deleted_at = 0;
CREATE INDEX IF NOT EXISTS idx_favorites_user ON favorites(user_id);
CREATE INDEX IF NOT EXISTS idx_simulation_sessions_user ON simulation_sessions(user_id, status);
CREATE INDEX IF NOT EXISTS idx_simulation_events_session ON simulation_events(session_id);
CREATE INDEX IF NOT EXISTS idx_point_transactions_user ON point_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_active ON subscriptions(user_id, status, ended_at);
CREATE INDEX IF NOT EXISTS idx_promo_codes_used_by ON promo_codes(used_by) WHERE used_by IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_promo_codes_partner ON promo_codes(partner_id);
CREATE INDEX IF NOT EXISTS idx_orders_user ON orders(user_id, status);
CREATE INDEX IF NOT EXISTS idx_user_activities_day_user ON user_activities(day, user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- +goose StatementEnd
