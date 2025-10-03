
DROP TABLE IF EXISTS stats CASCADE;

DROP INDEX IF EXISTS idx_login_history_ip_address;
DROP INDEX IF EXISTS idx_login_history_created_at;
DROP INDEX IF EXISTS idx_login_history_user_id;
DROP TABLE IF EXISTS login_history CASCADE;

DROP INDEX IF EXISTS idx_refresh_tokens_expires_at;
DROP INDEX IF EXISTS idx_refresh_tokens_token_family;
DROP INDEX IF EXISTS idx_refresh_tokens_token_hash;
DROP INDEX IF EXISTS idx_refresh_tokens_user_id;
DROP TABLE IF EXISTS refresh_tokens CASCADE;

DROP INDEX IF EXISTS idx_users_login;
DROP TABLE IF EXISTS users CASCADE;
