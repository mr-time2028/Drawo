package realtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	DrawOpStroke = "stroke"
	DrawOpErase  = "erase"
	DrawOpShape  = "shape"
	DrawOpFill   = "fill"
	DrawOpText   = "text"
	DrawOpClear  = "clear"
	DrawOpUndo   = "undo"
	DrawOpRedo   = "redo"

	ToolPencil      = "pencil"
	ToolBrush       = "brush"
	ToolMarker      = "marker"
	ToolCalligraphy = "calligraphy"

	ShapeLine      = "line"
	ShapeRectangle = "rectangle"
	ShapeEllipse   = "ellipse"
	ShapeTriangle  = "triangle"
	ShapeArrow     = "arrow"

	maxCanvasCoordinate = 4096
	maxStrokePoints     = 256
	maxBrushSize        = 64
	maxTextLength       = 60
	maxTextFontSize     = 128
	maxCanvasHistoryOps = 2000
	maxRedoOpsPerClient = 100

	// Drawing-specific anti-abuse limits. These are separate from the generic
	// WebSocket message limit because drawing/fill/clear operations can be much
	// more expensive or disruptive than chat/game control frames.
	maxDrawOpsPerSecond  = 30
	maxFillOpsPerSecond  = 8
	maxClearOpsPerMinute = 3
)

var hexColorRE = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Point represents a normalized canvas coordinate.
//
// Coordinates are validated on the server to avoid abusive payloads with huge
// values. The frontend can still scale these coordinates to the actual canvas
// size it renders.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// DrawOperation is the drawing protocol.
//
// It is intentionally operation-based, not image-based. Clients send compact
// operations (stroke, erase, shape, fill, undo, redo, clear), the room validates
// and orders them, then broadcasts the accepted operation to every player. New
// joiners receive the current operation history through canvas_sync.
type DrawOperation struct {
	Op string `json:"op"`

	// Server-authoritative metadata added by the room before broadcasting.
	ID        string `json:"id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	ServerSeq int64  `json:"server_seq,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`

	// Undo/redo metadata.
	TargetID string         `json:"target_id,omitempty"`
	Target   *DrawOperation `json:"target,omitempty"`

	// Stroke/erase fields.
	Tool   string  `json:"tool,omitempty"`
	Color  string  `json:"color,omitempty"`
	Size   float64 `json:"size,omitempty"`
	Points []Point `json:"points,omitempty"`

	// Opacity in (0,1]. Zero means "not set" and renders as fully opaque,
	// which keeps the field backward compatible with older clients.
	Opacity float64 `json:"opacity,omitempty"`

	// Shape/fill fields.
	Shape  string  `json:"shape,omitempty"`
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	Width  float64 `json:"width,omitempty"`
	Height float64 `json:"height,omitempty"`
	Filled bool    `json:"filled,omitempty"`

	// Text tool fields. Text content is validated for length here and checked
	// against the current word + bad-word list in the room hot path (the room
	// owns that context, not this stateless validator).
	Text     string  `json:"text,omitempty"`
	FontSize float64 `json:"font_size,omitempty"`
}

// CanvasSyncPayload is sent to a newly joined client so it can reconstruct the
// current round canvas without receiving a bitmap/image from the server.
type CanvasSyncPayload struct {
	Operations []DrawOperation `json:"operations"`
	ServerSeq  int64           `json:"server_seq"`
}

// ValidateDrawingPayload parses and validates a client drawing operation.
// It rejects malformed/oversized/unknown operations before they enter the room
// goroutine, which protects the realtime hot path from abusive WebSocket input.
func ValidateDrawingPayload(payload json.RawMessage) (DrawOperation, error) {
	if len(payload) == 0 {
		return DrawOperation{}, errors.New("drawing payload is required")
	}
	var op DrawOperation
	if err := json.Unmarshal(payload, &op); err != nil {
		return DrawOperation{}, errors.New("invalid drawing payload")
	}

	switch op.Op {
	case DrawOpStroke:
		return op, validateStroke(op)
	case DrawOpErase:
		return op, validateErase(op)
	case DrawOpShape:
		return op, validateShape(op)
	case DrawOpFill:
		return op, validateFill(op)
	case DrawOpText:
		return op, validateText(op)
	case DrawOpClear, DrawOpUndo, DrawOpRedo:
		return op, nil
	default:
		return DrawOperation{}, fmt.Errorf("unsupported drawing op: %s", op.Op)
	}
}

func validateStroke(op DrawOperation) error {
	if !validTool(op.Tool) {
		return errors.New("invalid stroke tool")
	}
	if !validColor(op.Color) {
		return errors.New("invalid stroke color")
	}
	if !validSize(op.Size) {
		return errors.New("invalid stroke size")
	}
	if !validOpacity(op.Opacity) {
		return errors.New("invalid stroke opacity")
	}
	return validatePoints(op.Points, 2)
}

func validateErase(op DrawOperation) error {
	if !validSize(op.Size) {
		return errors.New("invalid eraser size")
	}
	return validatePoints(op.Points, 1)
}

func validateShape(op DrawOperation) error {
	if !validShape(op.Shape) {
		return errors.New("invalid shape")
	}
	if !validColor(op.Color) {
		return errors.New("invalid shape color")
	}
	if !validOpacity(op.Opacity) {
		return errors.New("invalid shape opacity")
	}
	if !validCoordinate(op.X) || !validCoordinate(op.Y) {
		return errors.New("invalid shape coordinate")
	}
	if op.Shape == ShapeLine || op.Shape == ShapeArrow {
		// Lines/arrows store the endpoint in width/height (absolute coords).
		if !validCoordinate(op.Width) || !validCoordinate(op.Height) {
			return errors.New("invalid line endpoint")
		}
		return nil
	}
	if op.Width <= 0 || op.Height <= 0 || op.Width > maxCanvasCoordinate || op.Height > maxCanvasCoordinate {
		return errors.New("invalid shape dimensions")
	}
	return nil
}

func validateFill(op DrawOperation) error {
	if !validColor(op.Color) {
		return errors.New("invalid fill color")
	}
	if !validCoordinate(op.X) || !validCoordinate(op.Y) {
		return errors.New("invalid fill coordinate")
	}
	return nil
}

func validateText(op DrawOperation) error {
	trimmed := strings.TrimSpace(op.Text)
	if trimmed == "" {
		return errors.New("text is required")
	}
	if len([]rune(op.Text)) > maxTextLength {
		return errors.New("text too long")
	}
	if !validColor(op.Color) {
		return errors.New("invalid text color")
	}
	if !validOpacity(op.Opacity) {
		return errors.New("invalid text opacity")
	}
	if !validCoordinate(op.X) || !validCoordinate(op.Y) {
		return errors.New("invalid text coordinate")
	}
	if op.FontSize <= 0 || op.FontSize > maxTextFontSize {
		return errors.New("invalid text font size")
	}
	return nil
}

func validatePoints(points []Point, min int) error {
	if len(points) < min {
		return errors.New("not enough points")
	}
	if len(points) > maxStrokePoints {
		return errors.New("too many points")
	}
	for _, p := range points {
		if !validCoordinate(p.X) || !validCoordinate(p.Y) {
			return errors.New("point out of canvas bounds")
		}
	}
	return nil
}

func validTool(tool string) bool {
	switch tool {
	case ToolPencil, ToolBrush, ToolMarker, ToolCalligraphy:
		return true
	default:
		return false
	}
}

func validShape(shape string) bool {
	switch shape {
	case ShapeLine, ShapeRectangle, ShapeEllipse, ShapeTriangle, ShapeArrow:
		return true
	default:
		return false
	}
}

// validOpacity accepts the zero value ("not set" → opaque) or (0,1].
func validOpacity(opacity float64) bool {
	return opacity == 0 || (opacity > 0 && opacity <= 1)
}

func validColor(color string) bool {
	return hexColorRE.MatchString(color)
}

func validSize(size float64) bool {
	return size > 0 && size <= maxBrushSize
}

func validCoordinate(v float64) bool {
	return v >= 0 && v <= maxCanvasCoordinate
}
