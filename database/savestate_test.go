package database

import (
	"encoding/json"
	"testing"
)

// gameSaveKeys is every key the Godot client reads back from /game/getsavedata
// (soln-game helpers/save_state.gd, _http_request_completed), with the JSON type it assigns it into.
// The game fails on a missing key, and assigns into typed vars, so both have to match.
var gameSaveKeys = map[string]string{
	"current_floor":                     "number",
	"current_quest":                     "string",
	"saved_scene":                       "string",
	"vector_x":                          "number",
	"vector_y":                          "number",
	"first_time_init_floor1":            "bool",
	"first_time_init_floor2":            "bool",
	"first_time_init_floor3":            "bool",
	"rock_removed":                      "bool",
	"disable_rock_removed":              "bool",
	"raket_sneaking_quest_complete":     "bool",
	"unlock_cave_collision":             "bool",
	"raket_sword_complete":              "bool",
	"raket_quest_progress":              "number",
	"do_raket_blacksmith_animation":     "bool",
	"sword_bottom":                      "bool",
	"sword_guard":                       "bool",
	"sword_lower_blade":                 "bool",
	"sword_middle_blade":                "bool",
	"sword_top_blade":                   "bool",
	"disable_dead_robot_quest":          "bool",
	"disable_raket_stealing_quest":      "bool",
	"disable_fresh_dialogue_quest":      "bool",
	"disable_water_logged_1_quest":      "bool",
	"disable_water_logged_2_quest":      "bool",
	"disable_water_logged_3_quest":      "bool",
	"disable_chip_quest":                "bool",
	"disable_rat_wizard_training_quest": "bool",
	"player_badges":                     "object",
}

// gameBadgeKeys matches soln-game helpers/player_state.gd's player_badges dictionary.
var gameBadgeKeys = []string{
	"shiny_rock", "bowl", "carrot", "cake", "sword", "mushroom", "bucket1",
	"flask", "bucket2", "bucket3", "crystal_ball", "shell", "original_robot",
}

func jsonType(v any) string {
	switch v.(type) {
	case bool:
		return "bool"
	case float64:
		return "number"
	case string:
		return "string"
	case map[string]any:
		return "object"
	default:
		return "other"
	}
}

func TestDefaultSaveMatchesGameContract(t *testing.T) {
	var save map[string]any
	if err := json.Unmarshal([]byte(DefaultSave), &save); err != nil {
		t.Fatalf("DefaultSave isn't a JSON object: %v", err)
	}

	for key, want := range gameSaveKeys {
		v, ok := save[key]
		if !ok {
			t.Errorf("DefaultSave is missing %q, which the game reads on load", key)
			continue
		}
		if got := jsonType(v); got != want {
			t.Errorf("DefaultSave[%q] is a %s, want %s", key, got, want)
		}
	}
	for key := range save {
		if _, ok := gameSaveKeys[key]; !ok {
			t.Errorf("DefaultSave has %q, which the game never reads", key)
		}
	}

	badges, _ := save["player_badges"].(map[string]any)
	if len(badges) != len(gameBadgeKeys) {
		t.Errorf("player_badges has %d keys, want %d", len(badges), len(gameBadgeKeys))
	}
	for _, key := range gameBadgeKeys {
		if v, ok := badges[key]; !ok || jsonType(v) != "bool" {
			t.Errorf("player_badges[%q] = %v, want a bool", key, v)
		}
	}
}
