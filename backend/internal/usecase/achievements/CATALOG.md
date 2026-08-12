# Каталог ачивок КиноКруга

Файл сгенерирован из `catalog.go`. Ручные изменения будут перезаписаны.

| # | Код | Название | Условие | Метрика | Порог | XP | Политика | Секретная |
| ---: | --- | --- | --- | --- | ---: | ---: | --- | :---: |
| 1 | `first_frame` | Первый кадр | Оценить первый тайтл | `ratings_total` | 1 | 50 | `lifetime` | нет |
| 2 | `warmup` | Разогрев | Оценить 5 тайтлов | `ratings_total` | 5 | 50 | `lifetime` | нет |
| 3 | `solid_ten` | Крепкая десятка | Оценить 10 тайтлов | `ratings_total` | 10 | 100 | `lifetime` | нет |
| 4 | `movie_marathon` | Киномарафон | Оценить 25 тайтлов | `ratings_total` | 25 | 200 | `lifetime` | нет |
| 5 | `full_shelf` | Полная полка | Оценить 50 тайтлов | `ratings_total` | 50 | 350 | `lifetime` | нет |
| 6 | `hundred_credits` | Сотня в титрах | Оценить 100 тайтлов | `ratings_total` | 100 | 500 | `lifetime` | нет |
| 7 | `thoughtful_viewer` | Вдумчивый зритель | Оценить тайтл по всем 7 критериям | `full_ratings_total` | 1 | 50 | `lifetime` | нет |
| 8 | `broken_down` | По полочкам | Поставить 10 полных оценок по 7 критериям | `full_ratings_total` | 10 | 200 | `lifetime` | нет |
| 9 | `genre_loyalty` | Верность жанру | Оценить 10 тайтлов одного жанра | `ratings_same_genre_max` | 10 | 200 | `lifetime` | нет |
| 10 | `time_machine` | Машина времени | Оценить тайтлы из 5 разных десятилетий | `rated_release_decades_total` | 5 | 200 | `lifetime` | нет |
| 11 | `love_and_hate` | Между любовью и ненавистью | Иметь оценки не выше 3 и не ниже 9 | `rating_score_contrast` | 1 | 100 | `lifetime` | да |
| 12 | `absolute_ten` | Абсолютная десятка | Поставить по 10 по всем 7 критериям | `perfect_seven_rating_exists` | 1 | 350 | `lifetime` | да |
| 13 | `two_screens` | Два экрана | Оценить не менее 5 фильмов и 5 сериалов | `rated_media_balance` | 5 | 100 | `lifetime` | нет |
| 14 | `big_screen` | Большой экран | Оценить 20 фильмов | `rated_movies_total` | 20 | 200 | `lifetime` | нет |
| 15 | `one_more_episode` | Ещё одну серию | Оценить 20 сериалов | `rated_tv_total` | 20 | 200 | `lifetime` | нет |
| 16 | `omnivore` | Всеядный | Охватить 10 разных жанров | `rated_genres_total` | 10 | 200 | `lifetime` | нет |
| 17 | `three_evenings` | Три вечера подряд | Оценивать тайтлы 3 календарных дня подряд | `rating_day_streak` | 3 | 100 | `lifetime` | нет |
| 18 | `week_in_cinema` | Неделя в кино | Оценивать тайтлы 7 календарных дней подряд | `rating_day_streak` | 7 | 200 | `lifetime` | нет |
| 19 | `first_in_circle` | Первый в круге | Завести первого друга | `friends_high_water` | 1 | 50 | `lifetime` | нет |
| 20 | `own_company` | Своя компания | Достичь 5 друзей | `friends_high_water` | 5 | 200 | `lifetime` | нет |
| 21 | `shared_screen` | Общий сеанс | Оценить хотя бы один тайтл, оценённый другом | `shared_titles_total` | 1 | 50 | `lifetime` | нет |
| 22 | `cinema_club` | Киноклуб | Иметь 10 общих оценённых тайтлов с друзьями | `shared_titles_total` | 10 | 200 | `lifetime` | нет |
| 23 | `same_wavelength` | На одной волне | На 5 общих тайтлах отличаться с одним другом не больше чем на 0,5 | `friend_close_ratings_max` | 5 | 200 | `lifetime` | нет |
| 24 | `synchronized_take` | Синхронный дубль | Полностью совпасть с другом по всем 7 критериям одного тайтла | `friend_exact_full_rating_exists` | 1 | 350 | `lifetime` | да |
| 25 | `voice_from_the_audience` | Реплика из зала | Оставить первый комментарий | `comments_total` | 1 | 50 | `lifetime` | нет |
| 26 | `something_to_say` | Есть что сказать | Комментировать 5 разных тайтлов | `commented_titles_total` | 5 | 100 | `lifetime` | нет |
| 27 | `in_dialogue` | В диалоге | Написать 5 ответов на комментарии | `replies_authored_total` | 5 | 100 | `lifetime` | нет |
| 28 | `conversation_started` | Завязалась беседа | Получить ответ другого пользователя на свой комментарий | `received_reply_exists` | 1 | 100 | `lifetime` | нет |
| 29 | `noted` | На заметку | Добавить первый тайтл в список «Хочу посмотреть» | `watchlist_high_water` | 1 | 50 | `lifetime` | нет |
| 30 | `evening_supply` | Запас на вечер | Одновременно собрать 10 тайтлов в списке «Хочу посмотреть» | `watchlist_high_water` | 10 | 100 | `lifetime` | нет |
| 31 | `archivist` | Архивариус | Оценить 5 тайтлов, выпущенных до 1980 года | `rated_before_1980_total` | 5 | 100 | `lifetime` | нет |
| 32 | `classics_never_age` | Классика не стареет | Оценить 15 тайтлов, выпущенных до 2000 года | `rated_before_2000_total` | 15 | 200 | `lifetime` | нет |
| 33 | `genre_atlas` | Жанровый атлас | Охватить 15 разных жанров | `rated_genres_total` | 15 | 350 | `lifetime` | нет |
| 34 | `seven_facets` | Семь граней | Поставить 25 полных оценок по 7 критериям | `full_ratings_total` | 25 | 350 | `lifetime` | нет |
| 35 | `second_hundred` | Вторая сотня | Оценить 200 уникальных тайтлов | `ratings_total` | 200 | 500 | `lifetime` | нет |
| 36 | `spectrum` | Спектр | В одной полной оценке использовать 7 разных значений | `full_rating_distinct_score_max` | 7 | 200 | `lifetime` | да |
| 37 | `unanimity` | Единодушие | В одной полной оценке поставить одинаковый балл по всем критериям | `full_rating_same_score_exists` | 1 | 100 | `lifetime` | да |
| 38 | `triple_screening` | Тройной сеанс | Оценить 3 разных тайтла за один календарный день | `rating_same_day_max` | 3 | 100 | `lifetime` | да |
| 39 | `following_a_friend` | По следам друга | Оценить 5 тайтлов после того, как их оценил текущий друг | `friend_earlier_titles_total` | 5 | 200 | `lifetime` | нет |
| 40 | `pioneer` | Первооткрыватель | Первым оценить 5 тайтлов, которые позже оценили текущие друзья | `user_earlier_titles_total` | 5 | 200 | `lifetime` | нет |
| 41 | `three_seats` | Три места рядом | Найти тайтл, оценённый вами и двумя друзьями | `title_friend_raters_max` | 2 | 100 | `lifetime` | нет |
| 42 | `full_house` | Полный зал | Найти тайтл, оценённый вами и четырьмя друзьями | `title_friend_raters_max` | 4 | 350 | `lifetime` | нет |
| 43 | `wide_circle` | Широкий круг | Иметь общий оценённый тайтл с тремя разными друзьями | `friends_with_shared_title_total` | 3 | 200 | `lifetime` | нет |
| 44 | `opposites_attract` | Противоположности притягиваются | На 5 общих тайтлах отличаться с одним другом не меньше чем на 3 | `friend_far_ratings_max` | 5 | 200 | `lifetime` | да |
| 45 | `choir` | Хор | Получить одинаковую итоговую оценку одного тайтла у себя и двух друзей | `title_same_avg_friend_count_max` | 2 | 200 | `lifetime` | да |
| 46 | `discussion_regular` | Завсегдатай обсуждений | Комментировать 10 разных тайтлов | `commented_titles_total` | 10 | 200 | `lifetime` | нет |
| 47 | `critic_with_explanation` | Критик с пояснением | И оценить, и прокомментировать 10 одинаковых тайтлов | `rated_and_commented_titles_total` | 10 | 200 | `lifetime` | нет |
| 48 | `echo` | Эхо | Получить ответы от трёх разных пользователей | `distinct_repliers_total` | 3 | 200 | `lifetime` | нет |
| 49 | `hot_thread` | Горячая ветка | Получить под своим корневым комментарием 5 прямых ответов хотя бы от двух пользователей | `root_direct_replies_max` | 5 | 350 | `lifetime` | да |
| 50 | `matching_plans` | Планы совпали | Одновременно иметь один и тот же тайтл в списке «Хочу посмотреть» с другом | `friend_watchlist_match_exists` | 1 | 100 | `lifetime` | да |
| 51 | `no_weak_links` | Без слабых мест | Поставить 3 полные оценки, в каждой все критерии не ниже 8 | `no_weak_links_total` | 3 | 200 | `since_introduction` | нет |
| 52 | `no_concessions` | Без поблажек | Поставить 3 полные оценки, в каждой все критерии не выше 5 | `no_concessions_total` | 3 | 200 | `since_introduction` | нет |
| 53 | `contrast_cut` | Контрастный монтаж | В одной полной оценке одновременно поставить 1 и 10 при среднем балле от 4 до 7 | `contrast_cut_exists` | 1 | 350 | `since_introduction` | да |
| 54 | `signature_touch` | Фирменный почерк | В 3 полных оценках сделать один критерий единственным самым высоким | `signature_touch_max` | 3 | 200 | `since_introduction` | нет |
| 55 | `three_eras_one_day` | Три эпохи за вечер | За один день оценить 3 тайтла из 3 разных десятилетий | `three_eras_same_day_max` | 3 | 200 | `since_introduction` | нет |
| 56 | `genre_through_ages` | Жанр сквозь эпохи | Оценить тайтлы одного жанра из 4 разных десятилетий | `genre_decades_max` | 4 | 350 | `since_introduction` | нет |
| 57 | `parallel_years` | Параллельный эфир | Для 2 разных годов оценить и фильм, и сериал | `parallel_years_total` | 2 | 200 | `since_introduction` | нет |
| 58 | `five_notches` | Пять делений | Полными оценками закрыть 5 разных округлённых баллов, включая низкий и высокий | `five_notches_total` | 5 | 350 | `since_introduction` | нет |
| 59 | `middle_ground` | Третейский судья | Оказаться посередине между сильно различающимися оценками двух друзей | `middle_ground_exists` | 1 | 500 | `since_introduction` | да |
| 60 | `lone_dissenter` | Особое мнение | Сильно разойтись с двумя друзьями, которые оценили тайтл почти одинаково | `lone_dissenter_exists` | 1 | 500 | `since_introduction` | да |
| 61 | `together_and_apart` | И вместе, и врозь | С одним другом дважды почти совпасть и дважды сильно разойтись | `together_and_apart_max` | 2 | 350 | `since_introduction` | да |
| 62 | `rated_round_table` | Круглый стол | Вместе с двумя друзьями оценить и прокомментировать один тайтл | `rated_round_table_exists` | 1 | 350 | `since_introduction` | нет |
| 63 | `critic_duet` | Критический дуэт | С одним другом обоим оценить и прокомментировать 2 одинаковых тайтла | `critic_duet_total` | 2 | 350 | `since_introduction` | нет |
| 64 | `after_credits` | После титров | Прокомментировать 2 тайтла минимум через 48 часов после своей оценки | `after_credits_total` | 2 | 200 | `since_introduction` | нет |
| 65 | `council_watchlist` | Совет круга | Добавить один тайтл в список «Хочу посмотреть» вместе с двумя друзьями | `council_watchlist_max` | 2 | 350 | `since_introduction` | да |
| 66 | `opening_night` | Премьера в кругу | С другом оценить один тайтл с разницей не больше 4 часов | `opening_night_exists` | 1 | 350 | `since_introduction` | да |
| 67 | `chain_reaction` | Цепная реакция | После вашей оценки получить оценки того же тайтла от двух друзей за 72 часа | `chain_reaction_max` | 2 | 350 | `since_introduction` | нет |
| 68 | `trusted_recommendation` | Совет принят | Быстро добавить высоко оценённый другом тайтл и тоже высоко его оценить | `trusted_recommendation_exists` | 1 | 350 | `since_introduction` | нет |
| 69 | `changed_mind` | Переубедили | После ответа друга изменить низкую оценку тайтла минимум до 7 | `changed_mind_exists` | 1 | 500 | `since_introduction` | да |
| 70 | `patient_ticket` | Билет дождался | Оценить тайтл через 7–30 дней после добавления в список | `patient_ticket_exists` | 1 | 200 | `since_introduction` | нет |
| 71 | `clear_the_queue` | Разобрал очередь | За 48 часов оценить 2 тайтла из списка «Хочу посмотреть» | `clear_queue_max` | 2 | 200 | `since_introduction` | нет |
| 72 | `agreed_session` | Сеанс согласован | С другом почти одновременно добавить тайтл в список и обоим оценить его за 14 дней | `agreed_session_exists` | 1 | 350 | `since_introduction` | да |
| 73 | `relay` | Эстафета | Оценить тайтл между двумя друзьями с интервалами не больше 48 часов | `relay_exists` | 1 | 500 | `since_introduction` | да |
| 74 | `word_for_word` | Слово за слово | Обменяться с другом 4 чередующимися сообщениями в одной ветке за 48 часов | `word_for_word_max` | 4 | 200 | `since_introduction` | нет |
| 75 | `discuss_then_rate` | Сначала обсудили | С другом сначала обсудить тайтл, затем близко оценить его за 48 часов | `discuss_then_rate_exists` | 1 | 500 | `since_introduction` | да |
| 76 | `thread_resurrection` | Второй сезон | Оживить обсуждение старше 14 дней и получить ответ автора за 48 часов | `thread_resurrection_exists` | 1 | 350 | `since_introduction` | да |
| 77 | `two_good_tips` | Два верных совета | Высоко оценить два недавних совета от разных друзей | `good_tips_max` | 2 | 350 | `since_introduction` | нет |
| 78 | `mood_arc` | Сюжетный поворот | За 48 часов поставить низкую, среднюю и высокую оценки в таком порядке | `mood_arc_exists` | 1 | 350 | `since_introduction` | да |
| 79 | `deliberate_rating` | Выверенный вердикт | Начать с части критериев и завершить полную оценку через 12 часов–7 дней | `deliberate_rating_exists` | 1 | 200 | `since_introduction` | нет |
| 80 | `shared_finale` | Общий финал | В один день близко оценить тайтл вместе с двумя друзьями | `shared_finale_exists` | 1 | 350 | `since_introduction` | нет |
