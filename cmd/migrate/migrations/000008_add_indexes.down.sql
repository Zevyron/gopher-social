DROP INDEX IF EXISTS idx_comments_post_id;
DROP INDEX IF EXISTS idx_posts_user_id;
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_posts_tags;
DROP INDEX IF EXISTS idx_posts_title;
DROP INDEX IF EXISTS idx_comments_content;

-- Only drop the extension if you are absolutely sure no other tables 
-- or future migrations rely on it.
DROP EXTENSION IF EXISTS pg_trgm;