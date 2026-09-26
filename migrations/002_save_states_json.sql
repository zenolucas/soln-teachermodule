-- Migration 002: store each student's game save as one JSON document (SAVE-01).
--
-- Replaces save_states' ~45 fixed columns with `save_data JSON`. Saving deep-merges the posted
-- save into it (JSON_MERGE_PATCH), and loading merges it over database.DefaultSave, so adding a game
-- flag no longer needs a schema change, and a key the game doesn't send can't be reset.
--
-- Apply once, against a database set up from a soln_db.sql older than this migration (a fresh load
-- already has the new table):
--   mariadb -u$DB_USER -p$DB_PASSWORD $DB_NAME < migrations/002_save_states_json.sql
--
-- Existing rows are converted, not dropped. The game assigns these into typed bool vars, so every
-- flag goes through IF(col, true, false): a raw BOOLEAN column would come out as JSON 0/1, not true/false.
-- The FLOAT positions go through CAST(... AS DOUBLE): JSON_OBJECT renders a FLOAT with 6 significant
-- digits (1043.073 -> 1043.07), while DOUBLE carries the exact stored float32 value.

ALTER TABLE save_states
    ADD COLUMN IF NOT EXISTS save_data JSON NULL,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;

UPDATE save_states SET save_data = JSON_OBJECT(
    'current_floor', current_floor,
    'current_quest', current_quest,
    'saved_scene', saved_scene,
    'vector_x', CAST(vector_x AS DOUBLE),
    'vector_y', CAST(vector_y AS DOUBLE),
    'first_time_init_floor1', IF(first_time_init_floor1, true, false),
    'first_time_init_floor2', IF(first_time_init_floor2, true, false),
    'first_time_init_floor3', IF(first_time_init_floor3, true, false),
    'rock_removed', IF(rock_removed, true, false),
    'disable_rock_removed', IF(disable_rock_removed, true, false),
    'raket_sneaking_quest_complete', IF(raket_sneaking_quest_complete, true, false),
    'unlock_cave_collision', IF(unlock_cave_collision, true, false),
    'raket_sword_complete', IF(raket_sword_complete, true, false),
    'raket_quest_progress', raket_quest_progress,
    'do_raket_blacksmith_animation', IF(do_raket_blacksmith_animation, true, false),
    'sword_bottom', IF(sword_bottom, true, false),
    'sword_guard', IF(sword_guard, true, false),
    'sword_lower_blade', IF(sword_lower_blade, true, false),
    'sword_middle_blade', IF(sword_middle_blade, true, false),
    'sword_top_blade', IF(sword_top_blade, true, false),
    'disable_dead_robot_quest', IF(disable_dead_robot_quest, true, false),
    'disable_raket_stealing_quest', IF(disable_raket_stealing_quest, true, false),
    'disable_fresh_dialogue_quest', IF(disable_fresh_dialogue_quest, true, false),
    'disable_water_logged_1_quest', IF(disable_water_logged_1_quest, true, false),
    'disable_water_logged_2_quest', IF(disable_water_logged_2_quest, true, false),
    'disable_water_logged_3_quest', IF(disable_water_logged_3_quest, true, false),
    'disable_chip_quest', IF(disable_chip_quest, true, false),
    'disable_rat_wizard_training_quest', IF(disable_rat_wizard_training_quest, true, false),
    'player_badges', JSON_OBJECT(
        'shiny_rock', IF(badge_rock, true, false),
        'bowl', IF(badge_bowl, true, false),
        'carrot', IF(badge_carrot, true, false),
        'cake', IF(badge_cake, true, false),
        'sword', IF(badge_sword, true, false),
        'mushroom', IF(badge_mushroom, true, false),
        'bucket1', IF(badge_bucket1, true, false),
        'flask', IF(badge_flask, true, false),
        'bucket2', IF(badge_bucket2, true, false),
        'bucket3', IF(badge_bucket3, true, false),
        'crystal_ball', IF(badge_crystal_ball, true, false),
        'shell', IF(badge_shell, true, false),
        'original_robot', IF(badge_original_robot, true, false)
    )
) WHERE save_data IS NULL;

ALTER TABLE save_states MODIFY save_data JSON NOT NULL;

ALTER TABLE save_states
    DROP COLUMN current_floor, DROP COLUMN current_quest, DROP COLUMN saved_scene,
    DROP COLUMN vector_x, DROP COLUMN vector_y,
    DROP COLUMN first_time_init_floor1, DROP COLUMN first_time_init_floor2, DROP COLUMN first_time_init_floor3,
    DROP COLUMN rock_removed, DROP COLUMN disable_rock_removed, DROP COLUMN raket_sneaking_quest_complete,
    DROP COLUMN unlock_cave_collision, DROP COLUMN raket_sword_complete, DROP COLUMN raket_quest_progress,
    DROP COLUMN do_raket_blacksmith_animation,
    DROP COLUMN sword_bottom, DROP COLUMN sword_guard, DROP COLUMN sword_lower_blade,
    DROP COLUMN sword_middle_blade, DROP COLUMN sword_top_blade,
    DROP COLUMN disable_dead_robot_quest, DROP COLUMN disable_raket_stealing_quest,
    DROP COLUMN disable_fresh_dialogue_quest, DROP COLUMN disable_water_logged_1_quest,
    DROP COLUMN disable_water_logged_2_quest, DROP COLUMN disable_water_logged_3_quest,
    DROP COLUMN disable_chip_quest, DROP COLUMN disable_rat_wizard_training_quest,
    DROP COLUMN badge_rock, DROP COLUMN badge_bowl, DROP COLUMN badge_carrot, DROP COLUMN badge_cake,
    DROP COLUMN badge_sword, DROP COLUMN badge_mushroom, DROP COLUMN badge_bucket1, DROP COLUMN badge_flask,
    DROP COLUMN badge_bucket2, DROP COLUMN badge_bucket3, DROP COLUMN badge_crystal_ball,
    DROP COLUMN badge_shell, DROP COLUMN badge_original_robot;
