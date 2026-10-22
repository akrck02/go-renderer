package graphics

// CoordinateSpace says which space a shape's coordinates are in.
type CoordinateSpace int

// The spaces a shape can be given in.
const (
	// LocalSpace is the shape's own space, before the model matrix places it.
	LocalSpace CoordinateSpace = iota
	// WorldSpace is the scene's space, shared by every shape.
	WorldSpace
	// CameraSpace is the space seen from the camera, after the view matrix.
	CameraSpace
)
