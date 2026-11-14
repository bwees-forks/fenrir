package wolfapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Wolf rejects a session whose client_settings is missing any field.
func TestClientSettingsSerializesEveryWolfField(t *testing.T) {
	settings := ClientSettings{
		ControllersOverride:      []ControllerType{ControllerTypeXbox},
		MotionControllerOverride: ControllerTypeAuto,
		MouseAcceleration:        1,
		VScrollAcceleration:      1,
		HScrollAcceleration:      1,
		RunUID:                   1000,
		RunGID:                   1000,
	}

	encoded, err := json.Marshal(settings)
	require.NoError(t, err)

	assert.JSONEq(t, `{
		"controllers_override": ["XBOX"],
		"motion_controller_override": "AUTO",
		"mouse_acceleration": 1,
		"v_scroll_acceleration": 1,
		"h_scroll_acceleration": 1,
		"run_uid": 1000,
		"run_gid": 1000
	}`, string(encoded))
}
