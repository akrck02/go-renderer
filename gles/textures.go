package gles

import (
	"image"

	"github.com/akrck02/go-renderer/internal/sceneplan"
	"github.com/akrck02/go-renderer/scene"
)

// surfaceTextureUnit is the texture unit of the texture a draw lays over its surface.
const surfaceTextureUnit = 4

// textureStore keeps the scene's textures on the GPU, by picture: one upload for any number of
// meshes, and again only after the picture changed. It binds a texture only when the draw before
// had another. It belongs to one renderer, so a graphics context made again (a phone's
// application back from the background) starts with an empty store.
type textureStore struct {
	functions Functions
	byPicture map[uint64]*uploadedTexture
	bound     uint64 // the picture bound now, 0 = none since the frame began
	uploads   int    // pictures uploaded so far, which tests count
	packed    []byte // a picture's rows packed together, for pictures whose rows are not
}

// uploadedTexture is a picture on the GPU: which version of it, and how it is sampled.
type uploadedTexture struct {
	handle  Texture
	filled  bool // its pixels were uploaded once
	version uint64
	wrap    scene.Wrap
	sharp   bool
}

func newTextureStore(functions Functions) *textureStore {
	return &textureStore{functions: functions, byPicture: map[uint64]*uploadedTexture{}}
}

// beginFrame forgets what was bound: other passes may have used the unit since the last frame.
func (store *textureStore) beginFrame(program *program) {
	program.setInteger("surfaceTexture", surfaceTextureUnit)
	store.bound = 0
}

// setTexture tells the scene program whether a draw is textured, and how; it binds the texture.
func (store *textureStore) setTexture(program *program, material *scene.Material) {
	if !sceneplan.IsTextured(material) {
		program.setInteger("textured", 0)
		return
	}
	store.bind(material.Texture)
	program.setInteger("textured", 1)
	placement := sceneplan.TexturePlacement(material)
	program.setVector4("texturePlacement", placement)
}

// bind makes a texture the one draws read, uploading it first when new or changed.
func (store *textureStore) bind(texture *scene.Texture) {
	picture := texture.Picture
	uploaded := store.byPicture[picture.ID()]
	current := uploaded != nil && uploaded.version == picture.Version() && uploaded.wrap == texture.Wrap && uploaded.sharp == texture.Sharp
	if current && store.bound == picture.ID() {
		return
	}
	functions := store.functions
	functions.SelectTextureUnit(TextureUnit0 + surfaceTextureUnit)
	if !current {
		uploaded = store.upload(texture, uploaded)
	}
	functions.BindTexture(Texture2D, uploaded.handle)
	functions.SelectTextureUnit(TextureUnit0)
	store.bound = picture.ID()
}

// upload sends a texture's pixels (when new or changed) and its sampling to the GPU, into the
// texture it had if any. It runs only when something changed, never in a frame that only draws.
func (store *textureStore) upload(texture *scene.Texture, uploaded *uploadedTexture) *uploadedTexture {
	functions, picture := store.functions, texture.Picture
	if uploaded == nil {
		uploaded = &uploadedTexture{handle: functions.CreateTexture()}
		store.byPicture[picture.ID()] = uploaded
	}
	functions.BindTexture(Texture2D, uploaded.handle)
	if !uploaded.filled || uploaded.version != picture.Version() {
		width, height := picture.Size()
		functions.SetPixelStorage(UnpackAlignment, 1)
		functions.SetTextureImage(Texture2D, 0, RGBA, width, height, RGBA, UnsignedByte, store.rows(picture.Pixels))
		store.uploads++
	}
	setSampling(functions, texture)
	uploaded.filled, uploaded.version, uploaded.wrap, uploaded.sharp = true, picture.Version(), texture.Wrap, texture.Sharp
	return uploaded
}

// rows are a picture's pixels, row after row with nothing between: OpenGL ES 2 cannot skip the
// rest of a longer row.
func (store *textureStore) rows(pixels *image.NRGBA) []byte {
	width, height := pixels.Rect.Dx(), pixels.Rect.Dy()
	if pixels.Stride == 4*width {
		return pixels.Pix[:4*width*height]
	}
	store.packed = store.packed[:0]
	for row := range height {
		start := row * pixels.Stride
		store.packed = append(store.packed, pixels.Pix[start:start+4*width]...)
	}
	return store.packed
}

// glesWraps are the wrapping modes of scene.Wrap, in its order.
var glesWraps = [...]Enum{scene.Repeat: Repeat, scene.Clamp: ClampToEdge, scene.Mirror: MirroredRepeat}

// setSampling sets how the bound texture is read, as the texture asks. OpenGL ES 2 repeats and
// makes smaller copies of a picture only when its sides are powers of two: any other picture is
// clamped at its edges and read without them.
func setSampling(functions Functions, texture *scene.Texture) {
	wrap := glesWraps[scene.Repeat]
	if int(texture.Wrap) < len(glesWraps) {
		wrap = glesWraps[texture.Wrap]
	}
	width, height := texture.Picture.Size()
	powersOfTwo := isPowerOfTwo(width) && isPowerOfTwo(height)
	if !powersOfTwo {
		wrap = ClampToEdge
	}
	functions.SetTextureParameter(Texture2D, TextureWrapS, int32(wrap))
	functions.SetTextureParameter(Texture2D, TextureWrapT, int32(wrap))
	minifying, magnifying := LinearMipmapLinear, Linear
	switch {
	case texture.Sharp:
		minifying, magnifying = Nearest, Nearest
	case !powersOfTwo:
		minifying = Linear
	default:
		functions.GenerateMipmap(Texture2D)
	}
	functions.SetTextureParameter(Texture2D, TextureMinFilter, int32(minifying))
	functions.SetTextureParameter(Texture2D, TextureMagFilter, int32(magnifying))
}

func isPowerOfTwo(side int) bool { return side > 0 && side&(side-1) == 0 }
