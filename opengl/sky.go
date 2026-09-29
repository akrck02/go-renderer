package opengl

import (
	"image"
	"image/draw"
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// skyRenderer draws the sky dome (procedural or images) and the constellations.
type skyRenderer struct {
	program              *shaderProgram
	constellationProgram *shaderProgram
	emptyArray           uint32
	constellations       *constellationBuffers
}

// constellationBuffers holds the stars and lines of the constellations of one sky.
type constellationBuffers struct {
	sky                       *scene.Sky
	vertexArray, vertexBuffer uint32
	pointCount                int32
	lineRanges                []lineRange
}

// lineRange is the part of the line buffer of one constellation, with its color.
type lineRange struct {
	first, count int32
	color        graphics.Vec4
}

const (
	dayPanoramaUnit   = 3
	nightPanoramaUnit = 4
	dayCubeUnit       = 5
	nightCubeUnit     = 6
)

func newSkyRenderer() (*skyRenderer, error) {
	program, err := newShaderProgram(graphics.SkyVertexShader, graphics.SkyFragmentShader)
	if err != nil {
		return nil, err
	}
	constellationProgram, err := newShaderProgram(graphics.ConstellationVertexShader, graphics.ConstellationFragmentShader)
	if err != nil {
		return nil, err
	}
	renderer := &skyRenderer{program: program, constellationProgram: constellationProgram}
	gl.GenVertexArrays(1, &renderer.emptyArray)
	return renderer, nil
}

// draw paints the sky behind everything and leaves depth testing ready for the scene.
func (renderer *skyRenderer) draw(environment scene.Environment, light scene.Daylight, camera models.Camera, matrices cameraMatrices, seconds float64) {
	inverse, invertible := matrices.viewProjection.Inverse()
	if !invertible {
		return
	}
	gl.DepthMask(false)
	gl.Disable(gl.DEPTH_TEST)
	renderer.drawDome(environment, light, camera, inverse, seconds)
	renderer.drawConstellations(environment.EffectiveSky(), light, camera, matrices)
	gl.DepthMask(true)
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)
}

func (renderer *skyRenderer) drawDome(environment scene.Environment, light scene.Daylight, camera models.Camera, inverse graphics.Mat4, seconds float64) {
	sky := environment.EffectiveSky()
	program := renderer.program
	gl.UseProgram(program.handle)
	program.setMatrix("inverseViewProjection", inverse)
	program.setVector3("cameraPosition", camera.Position)
	program.setFloat("time", float32(seconds))
	setSunAndDaylight(program, environment, light)
	setStars(program, sky.Stars)
	setMoon(program, sky.Moon, light.MoonDirection)
	setClouds(program, sky.Clouds)
	setWind(program, environment.Wind)
	bindSkyImage(program, sky.DayImage, "dayImageKind", "dayPanorama", "dayCube", dayPanoramaUnit, dayCubeUnit)
	bindSkyImage(program, sky.NightImage, "nightImageKind", "nightPanorama", "nightCube", nightPanoramaUnit, nightCubeUnit)
	gl.BindVertexArray(renderer.emptyArray)
	gl.DrawArrays(gl.TRIANGLES, 0, 3)
}

func setSunAndDaylight(program *shaderProgram, environment scene.Environment, light scene.Daylight) {
	sky := environment.EffectiveSky()
	program.setVector3("sunDirection", environment.SunDirection.Normalize())
	program.setVector3("sunColor", environment.SunColor)
	program.setFloat("sunVisible", light.SunVisible)
	program.setVector3("skyColor", light.Zenith)
	program.setVector3("horizonColor", light.Horizon)
	program.setVector3("twilightColor", sky.TwilightColor)
	program.setFloat("dayAmount", light.DayAmount)
	program.setFloat("twilightAmount", light.TwilightAmount)
	program.setMatrix("starRotation", light.StarRotation)
}

func setStars(program *shaderProgram, stars scene.Stars) {
	program.setFloat("starDensity", stars.Density)
	program.setFloat("starBrightness", stars.Brightness)
	program.setFloat("starTwinkle", stars.Twinkle)
	program.setFloat("starDaytimeVisibility", stars.DaytimeVisibility)
}

func setMoon(program *shaderProgram, moon *scene.Moon, direction graphics.Vec4) {
	if moon == nil {
		program.setInteger("moonEnabled", 0)
		return
	}
	program.setInteger("moonEnabled", 1)
	program.setVector3("moonDirection", direction)
	program.setFloat("moonRadius", moon.AngularSize/2)
	program.setVector3("moonColor", moon.Color)
	program.setFloat("moonPhase", moon.Phase)
}

func setClouds(program *shaderProgram, clouds scene.Clouds) {
	program.setFloat("cloudCoverage", clouds.Coverage)
	program.setVector3("cloudColor", colorOrWhite(clouds.Color))
	program.setFloat("cloudSpeed", clouds.Speed)
	program.setFloat("cloudScale", scaleOrOne(clouds.Scale))
	gl.Uniform2f(program.location("cloudOffset"), clouds.Offset[0], clouds.Offset[2])
}

// setWind gives a program the horizontal wind direction and strength.
func setWind(program *shaderProgram, wind scene.Wind) {
	direction := graphics.Vec4{wind.Direction[0], 0, wind.Direction[2], 0}.Normalize()
	gl.Uniform2f(program.location("windDirection"), direction[0], direction[2])
	program.setFloat("windStrength", wind.Strength)
}

func colorOrWhite(color graphics.Vec4) graphics.Vec4 {
	if color == (graphics.Vec4{}) {
		return graphics.Vec4{1, 1, 1, 1}
	}
	return color
}

func scaleOrOne(value float32) float32 {
	if value <= 0 {
		return 1
	}
	return value
}

// bindSkyImage uploads the image once and binds it to its texture unit; kind tells the shader
// whether it is absent (0), a panorama (1) or a cube (2).
func bindSkyImage(program *shaderProgram, skyImage *scene.SkyImage, kindName, panoramaName, cubeName string, panoramaUnit, cubeUnit int32) {
	program.setInteger(panoramaName, panoramaUnit)
	program.setInteger(cubeName, cubeUnit)
	if skyImage == nil {
		program.setInteger(kindName, 0)
		return
	}
	texture := uploadSkyImage(skyImage)
	if skyImage.IsCube() {
		gl.ActiveTexture(gl.TEXTURE0 + uint32(cubeUnit))
		gl.BindTexture(gl.TEXTURE_CUBE_MAP, texture)
		program.setInteger(kindName, 2)
	} else {
		gl.ActiveTexture(gl.TEXTURE0 + uint32(panoramaUnit))
		gl.BindTexture(gl.TEXTURE_2D, texture)
		program.setInteger(kindName, 1)
	}
	gl.ActiveTexture(gl.TEXTURE0)
}

func uploadSkyImage(skyImage *scene.SkyImage) uint32 {
	if texture, uploaded := skyImage.GPU.(uint32); uploaded {
		return texture
	}
	var texture uint32
	if skyImage.IsCube() {
		texture = uploadCubeTexture(skyImage.Faces)
	} else {
		texture = uploadPanoramaTexture(skyImage.Equirectangular)
	}
	skyImage.GPU = texture
	return texture
}

func uploadPanoramaTexture(panorama image.Image) uint32 {
	var texture uint32
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	uploadImage(gl.TEXTURE_2D, panorama)
	gl.GenerateMipmap(gl.TEXTURE_2D)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.BindTexture(gl.TEXTURE_2D, 0)
	return texture
}

func uploadCubeTexture(faces [6]image.Image) uint32 {
	var texture uint32
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_CUBE_MAP, texture)
	for faceNumber, face := range faces {
		uploadImage(gl.TEXTURE_CUBE_MAP_POSITIVE_X+uint32(faceNumber), face)
	}
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	for _, wrap := range []uint32{gl.TEXTURE_WRAP_S, gl.TEXTURE_WRAP_T, gl.TEXTURE_WRAP_R} {
		gl.TexParameteri(gl.TEXTURE_CUBE_MAP, wrap, gl.CLAMP_TO_EDGE)
	}
	gl.BindTexture(gl.TEXTURE_CUBE_MAP, 0)
	return texture
}

// uploadImage sends an image as sRGB, so the shader samples linear colors.
func uploadImage(target uint32, source image.Image) {
	pixels := toRGBA(source)
	size := pixels.Bounds().Size()
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
	gl.TexImage2D(target, 0, gl.SRGB8_ALPHA8, int32(size.X), int32(size.Y), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels.Pix))
}

func toRGBA(source image.Image) *image.RGBA {
	if pixels, isRGBA := source.(*image.RGBA); isRGBA && pixels.Stride == pixels.Rect.Dx()*4 {
		return pixels
	}
	bounds := source.Bounds()
	pixels := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(pixels, pixels.Bounds(), source, bounds.Min, draw.Src)
	return pixels
}

// drawConstellations draws the lines and stars of every constellation, fading them with daylight.
func (renderer *skyRenderer) drawConstellations(sky *scene.Sky, light scene.Daylight, camera models.Camera, matrices cameraMatrices) {
	if len(sky.Constellations) == 0 {
		return
	}
	buffers := renderer.constellationBuffersFor(sky)
	darkness := float32(math.Pow(float64(1-light.DayAmount), 4))
	visibility := float32(math.Min(1, float64(darkness+sky.Stars.DaytimeVisibility*(1-darkness))))
	program := renderer.constellationProgram
	gl.UseProgram(program.handle)
	program.setMatrix("viewProjection", matrices.viewProjection)
	program.setVector3("cameraPosition", camera.Position)
	program.setMatrix("starRotation", light.StarRotation)
	program.setFloat("visibility", visibility)
	program.setFloat("pointSize", 5)
	enableBlending()
	gl.Enable(gl.PROGRAM_POINT_SIZE)
	gl.BindVertexArray(buffers.vertexArray)
	drawConstellationLines(program, buffers)
	drawConstellationStars(program, buffers, sky)
	gl.Disable(gl.PROGRAM_POINT_SIZE)
	gl.Disable(gl.BLEND)
	gl.BindVertexArray(0)
}

func drawConstellationLines(program *shaderProgram, buffers *constellationBuffers) {
	program.setInteger("drawingPoints", 0)
	for _, lines := range buffers.lineRanges {
		program.setVector4("lineColor", graphics.Vec4{lines.color[0], lines.color[1], lines.color[2], lines.color[3] * 0.55})
		gl.DrawArrays(gl.LINES, lines.first, lines.count)
	}
}

func drawConstellationStars(program *shaderProgram, buffers *constellationBuffers, sky *scene.Sky) {
	program.setInteger("drawingPoints", 1)
	program.setVector4("lineColor", graphics.Vec4{1, 1, 1, 1})
	gl.DrawArrays(gl.POINTS, 0, buffers.pointCount)
}

// constellationBuffersFor uploads the constellations of a sky the first time it is drawn.
func (renderer *skyRenderer) constellationBuffersFor(sky *scene.Sky) *constellationBuffers {
	if renderer.constellations != nil && renderer.constellations.sky == sky {
		return renderer.constellations
	}
	vertices, pointCount, lineRanges := constellationVertices(sky.Constellations)
	buffers := &constellationBuffers{sky: sky, pointCount: pointCount, lineRanges: lineRanges}
	gl.GenVertexArrays(1, &buffers.vertexArray)
	gl.BindVertexArray(buffers.vertexArray)
	gl.GenBuffers(1, &buffers.vertexBuffer)
	gl.BindBuffer(gl.ARRAY_BUFFER, buffers.vertexBuffer)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*bytesPerFloat, gl.Ptr(vertices), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, 3*bytesPerFloat, 0)
	gl.BindVertexArray(0)
	renderer.constellations = buffers
	return buffers
}

// constellationVertices lays out every star first (drawn as points) and then the line segments
// of each constellation.
func constellationVertices(constellations []scene.Constellation) (vertices []float32, pointCount int32, lineRanges []lineRange) {
	for _, constellation := range constellations {
		for _, star := range constellation.Stars {
			vertices = append(vertices, star[0], star[1], star[2])
			pointCount++
		}
	}
	next := pointCount
	for _, constellation := range constellations {
		first := next
		for _, line := range constellation.Lines {
			if !validLine(line, len(constellation.Stars)) {
				continue
			}
			for _, end := range line {
				star := constellation.Stars[end]
				vertices = append(vertices, star[0], star[1], star[2])
				next++
			}
		}
		lineRanges = append(lineRanges, lineRange{first: first, count: next - first, color: colorOrWhite(constellation.Color)})
	}
	return vertices, pointCount, lineRanges
}

func validLine(line [2]int, starCount int) bool {
	return line[0] >= 0 && line[1] >= 0 && line[0] < starCount && line[1] < starCount
}
