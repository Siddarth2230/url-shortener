CREATE TABLE IF NOT EXISTS urls (
    id BIGSERIAL PRIMARY KEY,
    short_code VARCHAR(10) UNIQUE NOT NULL,
    long_url TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS clicks (
    id BIGSERIAL PRIMARY KEY,
    short_code VARCHAR(10) REFERENCES urls(short_code) ON DELETE CASCADE,
    clicked_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    ip_address INET,
    user_agent TEXT,
    referer TEXT
);

CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code);

CREATE INDEX IF NOT EXISTS idx_clicks_short_code ON clicks(short_code);



