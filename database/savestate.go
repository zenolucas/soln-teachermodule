package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// DefaultSave is a new student's save, and the base every stored save is merged onto at load, so a
// key added later still loads for older saves. Its keys are the game's contract: the Godot client
// (soln-game helpers/save_state.gd) reads every one of them and fails on a missing key.
const DefaultSave = `{
	"current_floor": 1,
	"current_quest": "starting",
	"saved_scene": "res://scenes/levels/Floor1.tscn",
	"vector_x": 353,
	"vector_y": 163,
	"first_time_init_floor1": false,
	"first_time_init_floor2": false,
	"first_time_init_floor3": false,
	"rock_removed": false,
	"disable_rock_removed": false,
	"raket_sneaking_quest_complete": false,
	"unlock_cave_collision": false,
	"raket_sword_complete": false,
	"raket_quest_progress": 0,
	"do_raket_blacksmith_animation": false,
	"sword_bottom": false,
	"sword_guard": false,
	"sword_lower_blade": false,
	"sword_middle_blade": false,
	"sword_top_blade": false,
	"disable_dead_robot_quest": false,
	"disable_raket_stealing_quest": false,
	"disable_fresh_dialogue_quest": false,
	"disable_water_logged_1_quest": false,
	"disable_water_logged_2_quest": false,
	"disable_water_logged_3_quest": false,
	"disable_chip_quest": false,
	"disable_rat_wizard_training_quest": false,
	"player_badges": {
		"shiny_rock": false,
		"bowl": false,
		"carrot": false,
		"cake": false,
		"sword": false,
		"mushroom": false,
		"bucket1": false,
		"flask": false,
		"bucket2": false,
		"bucket3": false,
		"crystal_ball": false,
		"shell": false,
		"original_robot": false
	}
}`

// GetSaveData returns the student's save merged over DefaultSave, or DefaultSave itself if they
// have never saved.
func GetSaveData(ctx context.Context, studentID int) (json.RawMessage, error) {
	var doc string
	err := db.QueryRowContext(ctx,
		"SELECT JSON_MERGE_PATCH(?, save_data) FROM save_states WHERE student_id = ?",
		DefaultSave, studentID,
	).Scan(&doc)
	if errors.Is(err, sql.ErrNoRows) {
		return json.RawMessage(DefaultSave), nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetSaveData: %w", err)
	}
	return json.RawMessage(doc), nil
}

// MergeSaveData deep-merges patch (a JSON object) into the student's stored save, creating the row
// on the first save. Keys the patch leaves out keep their stored values (RFC 7396 merge), so a game
// build that doesn't send a key can't reset it.
func MergeSaveData(ctx context.Context, studentID int, patch []byte) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO save_states (student_id, save_data) VALUES (?, ?)
		ON DUPLICATE KEY UPDATE save_data = JSON_MERGE_PATCH(save_data, VALUES(save_data))
	`, studentID, patch)
	if err != nil {
		return fmt.Errorf("MergeSaveData: %w", err)
	}
	return nil
}
