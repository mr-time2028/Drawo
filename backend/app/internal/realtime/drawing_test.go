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
