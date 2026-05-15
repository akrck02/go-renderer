package models

type Renderer interface {
	DrawTriangle(points []float32, color int)
	FillTriangle(points []float32, color int)

	DrawRectangle(x int, y int, width int, height int, color int)
	FillRectangle(x int, y int, width int, height int, color int)

	DrawImage(bytes []byte, x int, y int, width int, height int)
}
