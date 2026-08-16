package realtime

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDrawingPayload(t *testing.T) {
	validStroke := json.RawMessage(`{"op":"stroke","tool":"pencil","color":"#112233","size":4,"points":[{"x":1,"y":2},{"x":3,"y":4}]}`)
	op, err := ValidateDrawingPayload(validStroke)
	require.NoError(t, err)
	assert.Equal(t, DrawOpStroke, op.Op)

	validErase := json.RawMessage(`{"op":"erase","size":20,"points":[{"x":1,"y":2}]}`)
	_, err = ValidateDrawingPayload(validErase)
	assert.NoError(t, err)

	validShape := json.RawMessage(`{"op":"shape","shape":"rectangle","color":"#abcdef","x":10,"y":20,"width":100,"height":80}`)
	_, err = ValidateDrawingPayload(validShape)
	assert.NoError(t, err)

	validLine := json.RawMessage(`{"op":"shape","shape":"line","color":"#abcdef","x":10,"y":20,"width":100,"height":80}`)
	_, err = ValidateDrawingPayload(validLine)
	assert.NoError(t, err)

	validFill := json.RawMessage(`{"op":"fill","color":"#abcdef","x":10,"y":20}`)
	_, err = ValidateDrawingPayload(validFill)
	assert.NoError(t, err)

	_, err = ValidateDrawingPayload(json.RawMessage(`{"op":"undo"}`))
	assert.NoError(t, err)
	_, err = ValidateDrawingPayload(json.RawMessage(`{"op":"redo"}`))
	assert.NoError(t, err)
	_, err = ValidateDrawingPayload(json.RawMessage(`{"op":"clear"}`))
	assert.NoError(t, err)

	// Arrow shape uses line-style endpoint semantics.
	validArrow := json.RawMessage(`{"op":"shape","shape":"arrow","color":"#abcdef","x":10,"y":20,"width":100,"height":80}`)
	_, err = ValidateDrawingPayload(validArrow)
	assert.NoError(t, err)

	// Opacity is accepted in (0,1] and as the zero value.
	validOpacity := json.RawMessage(`{"op":"stroke","tool":"marker","color":"#112233","size":4,"opacity":0.5,"points":[{"x":1,"y":2},{"x":3,"y":4}]}`)
	_, err = ValidateDrawingPayload(validOpacity)
	assert.NoError(t, err)

	// Text tool.
	validText := json.RawMessage(`{"op":"text","text":"hello","color":"#112233","x":10,"y":20,"font_size":24}`)
	op, err = ValidateDrawingPayload(validText)
	require.NoError(t, err)
	assert.Equal(t, DrawOpText, op.Op)
	assert.Equal(t, "hello", op.Text)
}

func TestValidateDrawingPayloadRejectsInvalidInput(t *testing.T) {
	cases := []json.RawMessage{
		json.RawMessage(``),
		json.RawMessage(`not-json`),
		json.RawMessage(`{"op":"unknown"}`),
		json.RawMessage(`{"op":"stroke","tool":"spray","color":"#112233","size":4,"points":[{"x":1,"y":2},{"x":3,"y":4}]}`),
		json.RawMessage(`{"op":"stroke","tool":"pencil","color":"red","size":4,"points":[{"x":1,"y":2},{"x":3,"y":4}]}`),
		json.RawMessage(`{"op":"stroke","tool":"pencil","color":"#112233","size":100,"points":[{"x":1,"y":2},{"x":3,"y":4}]}`),
		json.RawMessage(`{"op":"stroke","tool":"pencil","color":"#112233","size":4,"points":[{"x":1,"y":2}]}`),
		json.RawMessage(`{"op":"erase","size":20,"points":[{"x":99999,"y":2}]}`),
		json.RawMessage(`{"op":"shape","shape":"star","color":"#abcdef","x":10,"y":20,"width":100,"height":80}`),
		json.RawMessage(`{"op":"shape","shape":"rectangle","color":"#abcdef","x":10,"y":20,"width":0,"height":80}`),
		json.RawMessage(`{"op":"fill","color":"#abcdef","x":-1,"y":20}`),
		json.RawMessage(`{"op":"stroke","tool":"pencil","color":"#112233","size":4,"opacity":1.5,"points":[{"x":1,"y":2},{"x":3,"y":4}]}`),
		json.RawMessage(`{"op":"stroke","tool":"pencil","color":"#112233","size":4,"opacity":-0.2,"points":[{"x":1,"y":2},{"x":3,"y":4}]}`),
		json.RawMessage(`{"op":"text","text":"","color":"#112233","x":10,"y":20,"font_size":24}`),
		json.RawMessage(`{"op":"text","text":"hi","color":"#112233","x":10,"y":20,"font_size":0}`),
		json.RawMessage(`{"op":"text","text":"hi","color":"#112233","x":10,"y":20,"font_size":4096}`),
		json.RawMessage(`{"op":"text","text":"hi","color":"nope","x":10,"y":20,"font_size":24}`),
		json.RawMessage(`{"op":"shape","shape":"arrow","color":"#abcdef","x":10,"y":20,"width":99999,"height":80}`),
	}
	for _, tc := range cases {
		_, err := ValidateDrawingPayload(tc)
		assert.Error(t, err, string(tc))
	}
}

func TestValidateDrawingPayloadRejectsTooManyPoints(t *testing.T) {
	points := make([]Point, maxStrokePoints+1)
	for i := range points {
		points[i] = Point{X: float64(i), Y: 1}
	}
	payload, err := json.Marshal(DrawOperation{Op: DrawOpStroke, Tool: ToolPencil, Color: "#112233", Size: 4, Points: points})
	require.NoError(t, err)
	_, err = ValidateDrawingPayload(payload)
	assert.Error(t, err)
}
