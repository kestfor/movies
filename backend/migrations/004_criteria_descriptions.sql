ALTER TABLE criteria
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';

UPDATE criteria
SET description = CASE code
    WHEN 'plot' THEN 'Насколько история цельная, увлекательная и логичная.'
    WHEN 'directing' THEN 'Как собраны сцены, темп, акценты и общее авторское видение.'
    WHEN 'acting' THEN 'Насколько убедительно актёры передают персонажей и эмоции.'
    WHEN 'music' THEN 'Насколько музыка и звук помогают настроению и сценам.'
    WHEN 'visuals' THEN 'Качество картинки, света, цвета, эффектов и постановки кадра.'
    WHEN 'atmosphere' THEN 'Насколько сильно работа погружает в свой мир и настроение.'
    ELSE description
END
WHERE code IN ('plot', 'directing', 'acting', 'music', 'visuals', 'atmosphere');

---- create above / drop below ----

ALTER TABLE criteria
    DROP COLUMN IF EXISTS description;
