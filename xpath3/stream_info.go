package xpath3

// StreamInfo exposes precomputed streamability properties for use by
// internal packages (e.g. xslt3 streamability analysis). This struct
// is not intended for end-user consumption; it lives in an internal
// package that re-exports query helpers.
type StreamInfo struct {
	AxisUsed             uint16
	HasDownwardStep      bool
	HasDescOrSelf        bool
	HasNonMotionlessPred bool
	DownwardSelections   int
	UsedFunctions        map[string]bool
	IsContextItem        bool
}

// StreamInfo returns a snapshot of the precomputed streamability
// properties for this expression.
func (e *Expression) StreamInfo() StreamInfo {
	if e == nil || e.program == nil {
		return StreamInfo{}
	}
	s := e.program.stream
	return StreamInfo{
		AxisUsed:             s.axisUsed,
		HasDownwardStep:      s.hasDownwardStep,
		HasDescOrSelf:        s.hasDescOrSelf,
		HasNonMotionlessPred: s.hasNonMotionlessPred,
		DownwardSelections:   s.downwardSelections,
		UsedFunctions:        s.usedFunctions,
		IsContextItem:        s.isContextItem,
	}
}
