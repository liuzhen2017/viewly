-- 004: banner jump target — custom URL (link) takes priority; drama_id falls
-- back to jumping straight into that drama's first episode player.
ALTER TABLE banners ADD COLUMN link VARCHAR(500) NOT NULL DEFAULT '';
