UPDATE criteria
SET
    code = 'story',
    name = 'История',
    description = 'Насколько интересно следить за событиями, конфликтами и тем, к чему всё приходит.',
    sort_order = 1
WHERE code = 'plot';

INSERT INTO criteria (code, name, description, sort_order, is_active)
VALUES (
    'characters',
    'Герои',
    'Насколько живыми кажутся персонажи, их решения, отношения и изменения по ходу истории.',
    2,
    true
)
ON CONFLICT (code) DO UPDATE
SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    sort_order = EXCLUDED.sort_order,
    is_active = true;

UPDATE criteria
SET
    code = 'acting',
    name = 'Игра',
    description = 'Насколько веришь актёрам, их эмоциям, реакциям и тому, как они держат свои роли.',
    sort_order = 3
WHERE code = 'acting';

UPDATE criteria
SET
    code = 'direction',
    name = 'Подача',
    description = 'Насколько хорошо всё собрано: темп, сцены, акценты и ощущение цельности.',
    sort_order = 4
WHERE code = 'directing';

UPDATE criteria
SET
    code = 'visuals',
    name = 'Визуал',
    description = 'Насколько приятно и выразительно это выглядит: кадры, цвет, свет, костюмы и эффекты.',
    sort_order = 5
WHERE code = 'visuals';

UPDATE criteria
SET
    code = 'sound',
    name = 'Звук',
    description = 'Насколько музыка, звуки и тишина помогают сценам звучать сильнее.',
    sort_order = 6
WHERE code = 'music';

UPDATE criteria
SET
    code = 'atmosphere',
    name = 'Атмосфера',
    description = 'Насколько сильно фильм или сериал погружает в своё настроение и мир.',
    sort_order = 7
WHERE code = 'atmosphere';

---- create above / drop below ----

UPDATE criteria
SET
    code = 'plot',
    name = 'Сюжет',
    description = 'Насколько история цельная, увлекательная и логичная.',
    sort_order = 1
WHERE code = 'story';

UPDATE criteria
SET is_active = false
WHERE code = 'characters';

UPDATE criteria
SET
    code = 'directing',
    name = 'Режиссура',
    description = 'Как собраны сцены, темп, акценты и общее авторское видение.',
    sort_order = 2
WHERE code = 'direction';

UPDATE criteria
SET
    code = 'acting',
    name = 'Актёрская игра',
    description = 'Насколько убедительно актёры передают персонажей и эмоции.',
    sort_order = 3
WHERE code = 'acting';

UPDATE criteria
SET
    code = 'music',
    name = 'Музыка',
    description = 'Насколько музыка и звук помогают настроению и сценам.',
    sort_order = 4
WHERE code = 'sound';

UPDATE criteria
SET
    code = 'visuals',
    name = 'Визуал',
    description = 'Качество картинки, света, цвета, эффектов и постановки кадра.',
    sort_order = 5
WHERE code = 'visuals';

UPDATE criteria
SET
    name = 'Атмосфера',
    description = 'Насколько сильно работа погружает в свой мир и настроение.',
    sort_order = 6
WHERE code = 'atmosphere';
