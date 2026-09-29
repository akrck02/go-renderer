package loaders

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// Subset of the glTF 2.0 document used by LoadScene. Field names follow the specification.
type gltfDocument struct {
	Scene       int                        `json:"scene"`
	Scenes      []gltfSceneDefinition      `json:"scenes"`
	Nodes       []gltfNodeDefinition       `json:"nodes"`
	Meshes      []gltfMeshDefinition       `json:"meshes"`
	Materials   []gltfMaterialDefinition   `json:"materials"`
	Accessors   []gltfAccessorDefinition   `json:"accessors"`
	BufferViews []gltfBufferViewDefinition `json:"bufferViews"`
	Buffers     []gltfBufferDefinition     `json:"buffers"`
}

type gltfSceneDefinition struct {
	Nodes  []int          `json:"nodes"`
	Extras map[string]any `json:"extras"`
}

type gltfNodeDefinition struct {
	Name        string    `json:"name"`
	Mesh        *int      `json:"mesh"`
	Children    []int     `json:"children"`
	Matrix      []float32 `json:"matrix"`
	Translation []float32 `json:"translation"`
	Rotation    []float32 `json:"rotation"`
	Scale       []float32 `json:"scale"`
	Extensions  struct {
		Instancing *gltfInstancingExtension `json:"EXT_mesh_gpu_instancing"`
	} `json:"extensions"`
	Extras map[string]any `json:"extras"`
}

type gltfInstancingExtension struct {
	Attributes map[string]int `json:"attributes"`
}

type gltfMeshDefinition struct {
	Name       string                    `json:"name"`
	Primitives []gltfPrimitiveDefinition `json:"primitives"`
	Extras     map[string]any            `json:"extras"`
}

type gltfPrimitiveDefinition struct {
	Attributes map[string]int `json:"attributes"`
	Indices    *int           `json:"indices"`
	Material   *int           `json:"material"`
	Mode       *int           `json:"mode"`
	Extras     map[string]any `json:"extras"`
}

type gltfMaterialDefinition struct {
	Name                 string `json:"name"`
	PbrMetallicRoughness struct {
		BaseColorFactor []float32 `json:"baseColorFactor"`
	} `json:"pbrMetallicRoughness"`
	DoubleSided bool           `json:"doubleSided"`
	AlphaMode   string         `json:"alphaMode"`
	Extras      map[string]any `json:"extras"`
}

type gltfAccessorDefinition struct {
	BufferView    *int   `json:"bufferView"`
	ByteOffset    int    `json:"byteOffset"`
	ComponentType int    `json:"componentType"`
	Normalized    bool   `json:"normalized"`
	Count         int    `json:"count"`
	Type          string `json:"type"`
}

type gltfBufferViewDefinition struct {
	Buffer     int `json:"buffer"`
	ByteOffset int `json:"byteOffset"`
	ByteLength int `json:"byteLength"`
	ByteStride int `json:"byteStride"`
}

type gltfBufferDefinition struct {
	ByteLength int    `json:"byteLength"`
	URI        string `json:"uri"`
}

const (
	glbMagic          = 0x46546C67
	glbJSONChunk      = 0x4E4F534A
	glbBinaryChunk    = 0x004E4942
	trianglesMode     = 4
	componentByte     = 5120
	componentUnsigned = 5121
	componentShort    = 5122
	componentUShort   = 5123
	componentUInt     = 5125
	componentFloat    = 5126
)

var componentsPerElement = map[string]int{"SCALAR": 1, "VEC2": 2, "VEC3": 3, "VEC4": 4, "MAT4": 16}

var bytesPerComponent = map[int]int{componentByte: 1, componentUnsigned: 1, componentShort: 2, componentUShort: 2, componentUInt: 4, componentFloat: 4}

// gltfFile is a parsed document with its binary buffers.
type gltfFile struct {
	document  gltfDocument
	buffers   [][]byte
	directory string // folder of the file, for external buffers and sky images
}

// LoadScene reads a glTF 2.0 file (.glb, or .gltf with external buffers) into a scene.Scene.
//
// Supported: triangle primitives with POSITION, NORMAL, COLOR_0 and indices; node hierarchy with
// matrix or TRS; material base color, doubleSided and alphaMode; EXT_mesh_gpu_instancing
// (TRANSLATION, ROTATION, SCALE and a custom _COLOR). Extras understood by the renderer:
//   - scene extras "environment": sunDirection, sunColor, skyZenith, skyHorizon, groundAmbient,
//     ambient, fog {color, near, far}, water {level, deep, shallow, size, waveLength, floorDepth, colorDepth},
//     verticalScale, sky (see skyFromExtras)
//   - scene extras "environment" also: wind {direction, strength}
//   - material extras "kind": "lit" | "unlit" | "water" | "waterfall"; "sway": bending in the wind
//   - mesh, primitive or node extras "ground": true (walkable surface)
func LoadScene(path string) (*scene.Scene, error) {
	file, err := readGLTFFile(path)
	if err != nil {
		return nil, err
	}
	return file.buildScene()
}

func readGLTFFile(path string) (*gltfFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	file := &gltfFile{directory: filepath.Dir(path)}
	if isGLBContainer(content) {
		err = file.parseGLBContainer(content)
	} else {
		err = json.Unmarshal(content, &file.document)
	}
	if err != nil {
		return nil, err
	}
	return file, file.loadExternalBuffers(file.directory)
}

func isGLBContainer(content []byte) bool {
	return len(content) >= 12 && binary.LittleEndian.Uint32(content) == glbMagic
}

// parseGLBContainer reads the JSON chunk and the embedded binary chunk of a .glb file.
func (file *gltfFile) parseGLBContainer(content []byte) error {
	var jsonChunk, binaryChunk []byte
	for offset := 12; offset+8 <= len(content); {
		length := int(binary.LittleEndian.Uint32(content[offset:]))
		chunkType := binary.LittleEndian.Uint32(content[offset+4:])
		if offset+8+length > len(content) {
			return fmt.Errorf("glb: truncated chunk")
		}
		data := content[offset+8 : offset+8+length]
		switch chunkType {
		case glbJSONChunk:
			jsonChunk = data
		case glbBinaryChunk:
			binaryChunk = data
		}
		offset += 8 + length
	}
	file.buffers = [][]byte{binaryChunk}
	return json.Unmarshal(jsonChunk, &file.document)
}

// loadExternalBuffers reads buffers stored next to a .gltf file.
func (file *gltfFile) loadExternalBuffers(directory string) error {
	for bufferNumber, buffer := range file.document.Buffers {
		if buffer.URI == "" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, buffer.URI))
		if err != nil {
			return fmt.Errorf("gltf buffer %d: %w", bufferNumber, err)
		}
		for len(file.buffers) <= bufferNumber {
			file.buffers = append(file.buffers, nil)
		}
		file.buffers[bufferNumber] = data
	}
	return nil
}

// readAccessor returns the accessor values as floats and the number of components per element.
func (file *gltfFile) readAccessor(accessorNumber int) ([]float32, int, error) {
	if accessorNumber < 0 || accessorNumber >= len(file.document.Accessors) {
		return nil, 0, fmt.Errorf("accessor %d out of range", accessorNumber)
	}
	accessor := file.document.Accessors[accessorNumber]
	components := componentsPerElement[accessor.Type]
	componentSize := bytesPerComponent[accessor.ComponentType]
	if components == 0 || componentSize == 0 {
		return nil, 0, fmt.Errorf("accessor %d: unsupported %s of component type %d", accessorNumber, accessor.Type, accessor.ComponentType)
	}
	values := make([]float32, accessor.Count*components)
	if accessor.BufferView == nil {
		return values, components, nil
	}
	view := file.document.BufferViews[*accessor.BufferView]
	data := file.buffers[view.Buffer]
	stride := view.ByteStride
	if stride == 0 {
		stride = componentSize * components
	}
	start := view.ByteOffset + accessor.ByteOffset
	for element := 0; element < accessor.Count; element++ {
		for component := 0; component < components; component++ {
			offset := start + element*stride + component*componentSize
			if offset+componentSize > len(data) {
				return nil, 0, fmt.Errorf("accessor %d: out of buffer bounds", accessorNumber)
			}
			values[element*components+component] = readComponent(data[offset:], accessor.ComponentType, accessor.Normalized)
		}
	}
	return values, components, nil
}

// readComponent decodes one component, mapping normalized integers to [0, 1] or [-1, 1].
func readComponent(data []byte, componentType int, normalized bool) float32 {
	switch componentType {
	case componentFloat:
		return math.Float32frombits(binary.LittleEndian.Uint32(data))
	case componentUnsigned:
		return normalizeUnsigned(float32(data[0]), 255, normalized)
	case componentUShort:
		return normalizeUnsigned(float32(binary.LittleEndian.Uint16(data)), 65535, normalized)
	case componentByte:
		return normalizeSigned(float32(int8(data[0])), 127, normalized)
	case componentShort:
		return normalizeSigned(float32(int16(binary.LittleEndian.Uint16(data))), 32767, normalized)
	case componentUInt:
		return float32(binary.LittleEndian.Uint32(data))
	}
	return 0
}

func normalizeUnsigned(value, maximum float32, normalized bool) float32 {
	if normalized {
		return value / maximum
	}
	return value
}

func normalizeSigned(value, maximum float32, normalized bool) float32 {
	if normalized {
		return float32(math.Max(float64(value/maximum), -1))
	}
	return value
}

func (file *gltfFile) readIndices(accessorNumber int) ([]uint32, error) {
	values, _, err := file.readAccessor(accessorNumber)
	if err != nil {
		return nil, err
	}
	indices := make([]uint32, len(values))
	for position, value := range values {
		indices[position] = uint32(value)
	}
	return indices, nil
}

func (file *gltfFile) buildScene() (*scene.Scene, error) {
	result := scene.New()
	materials := file.buildMaterials()
	meshes, err := file.buildMeshes(materials)
	if err != nil {
		return nil, err
	}
	rootNodes, sceneExtras := file.rootNodesAndExtras()
	for _, nodeNumber := range rootNodes {
		node, err := file.buildNode(nodeNumber, meshes)
		if err != nil {
			return nil, err
		}
		result.Nodes = append(result.Nodes, node)
	}
	result.Tags = sceneExtras
	if environment, found := sceneExtras["environment"].(map[string]any); found {
		applyEnvironment(&result.Environment, environment)
		if err := file.applySky(&result.Environment, environment); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (file *gltfFile) rootNodesAndExtras() ([]int, map[string]any) {
	document := file.document
	if len(document.Scenes) > 0 && document.Scene < len(document.Scenes) {
		chosen := document.Scenes[document.Scene]
		return chosen.Nodes, chosen.Extras
	}
	all := make([]int, len(document.Nodes))
	for nodeNumber := range all {
		all[nodeNumber] = nodeNumber
	}
	return all, nil
}

func (file *gltfFile) buildMaterials() []*scene.Material {
	materials := make([]*scene.Material, len(file.document.Materials))
	for materialNumber, definition := range file.document.Materials {
		materials[materialNumber] = &scene.Material{
			Name:        definition.Name,
			Kind:        materialKindFromExtras(definition.Extras),
			BaseColor:   vectorOrDefault(definition.PbrMetallicRoughness.BaseColorFactor, graphics.Vec4{1, 1, 1, 1}),
			DoubleSided: definition.DoubleSided,
			Transparent: definition.AlphaMode == "BLEND",
			Sway:        extraNumber(definition.Extras, "sway", 0),
		}
	}
	return materials
}

func materialKindFromExtras(extras map[string]any) scene.MaterialKind {
	switch extras["kind"] {
	case "unlit":
		return scene.KindUnlit
	case "water":
		return scene.KindWater
	case "waterfall":
		return scene.KindWaterfall
	}
	return scene.KindLit
}

// buildMeshes returns, for every glTF mesh, one scene.Mesh per triangle primitive.
func (file *gltfFile) buildMeshes(materials []*scene.Material) ([][]*scene.Mesh, error) {
	meshes := make([][]*scene.Mesh, len(file.document.Meshes))
	for meshNumber, definition := range file.document.Meshes {
		for _, primitive := range definition.Primitives {
			mesh, err := file.buildPrimitive(definition, primitive, materials)
			if err != nil {
				return nil, err
			}
			if mesh != nil {
				meshes[meshNumber] = append(meshes[meshNumber], mesh)
			}
		}
	}
	return meshes, nil
}

func (file *gltfFile) buildPrimitive(mesh gltfMeshDefinition, primitive gltfPrimitiveDefinition, materials []*scene.Material) (*scene.Mesh, error) {
	positionAccessor, hasPositions := primitive.Attributes["POSITION"]
	if !hasPositions || (primitive.Mode != nil && *primitive.Mode != trianglesMode) {
		return nil, nil
	}
	positions, _, err := file.readAccessor(positionAccessor)
	if err != nil {
		return nil, err
	}
	result := &scene.Mesh{Name: mesh.Name, Positions: positions, Ground: isGround(mesh.Extras) || isGround(primitive.Extras)}
	if normalAccessor, found := primitive.Attributes["NORMAL"]; found {
		if result.Normals, _, err = file.readAccessor(normalAccessor); err != nil {
			return nil, err
		}
	}
	if colorAccessor, found := primitive.Attributes["COLOR_0"]; found {
		if result.Colors, err = file.readVertexColors(colorAccessor); err != nil {
			return nil, err
		}
	}
	if primitive.Indices != nil {
		if result.Indices, err = file.readIndices(*primitive.Indices); err != nil {
			return nil, err
		}
	}
	if primitive.Material != nil && *primitive.Material < len(materials) {
		result.Material = materials[*primitive.Material]
	}
	return result, nil
}

func isGround(extras map[string]any) bool { return extras["ground"] == true }

// readVertexColors reads COLOR_0 as RGBA, adding opaque alpha to RGB colors.
func (file *gltfFile) readVertexColors(accessorNumber int) ([]float32, error) {
	colors, components, err := file.readAccessor(accessorNumber)
	if err != nil || components == 4 {
		return colors, err
	}
	return expandRGBToRGBA(colors), nil
}

func expandRGBToRGBA(rgb []float32) []float32 {
	vertexCount := len(rgb) / 3
	rgba := make([]float32, vertexCount*4)
	for vertex := 0; vertex < vertexCount; vertex++ {
		copy(rgba[4*vertex:4*vertex+3], rgb[3*vertex:3*vertex+3])
		rgba[4*vertex+3] = 1
	}
	return rgba
}

func (file *gltfFile) buildNode(nodeNumber int, meshes [][]*scene.Mesh) (*scene.Node, error) {
	definition := file.document.Nodes[nodeNumber]
	node := &scene.Node{Name: definition.Name, Transform: nodeTransform(definition), Tags: definition.Extras}
	instances, err := file.readInstances(definition.Extensions.Instancing)
	if err != nil {
		return nil, err
	}
	if definition.Mesh != nil && *definition.Mesh < len(meshes) {
		attachPrimitives(node, meshes[*definition.Mesh], instances, isGround(definition.Extras))
	}
	for _, childNumber := range definition.Children {
		if childNumber < 0 || childNumber >= len(file.document.Nodes) {
			continue
		}
		child, err := file.buildNode(childNumber, meshes)
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, child)
	}
	return node, nil
}

func nodeTransform(definition gltfNodeDefinition) graphics.Mat4 {
	if len(definition.Matrix) == 16 {
		var matrix graphics.Mat4
		copy(matrix[:], definition.Matrix)
		return matrix
	}
	return graphics.FromTranslationRotationScale(
		vectorOrDefault(definition.Translation, graphics.Vec4{}),
		vectorOrDefault(definition.Rotation, graphics.Vec4{0, 0, 0, 1}),
		vectorOrDefault(definition.Scale, graphics.Vec4{1, 1, 1, 1}),
	)
}

// attachPrimitives puts the first primitive on the node and the others on child nodes.
func attachPrimitives(node *scene.Node, primitives []*scene.Mesh, instances *scene.Instances, ground bool) {
	for primitiveNumber, mesh := range primitives {
		mesh.Ground = mesh.Ground || ground
		if primitiveNumber == 0 {
			node.Mesh, node.Instances = mesh, instances
			continue
		}
		node.Children = append(node.Children, &scene.Node{Name: node.Name, Transform: graphics.Identity(), Mesh: mesh, Instances: instances})
	}
}

// readInstances builds the instance transforms and colors of EXT_mesh_gpu_instancing.
func (file *gltfFile) readInstances(extension *gltfInstancingExtension) (*scene.Instances, error) {
	if extension == nil {
		return nil, nil
	}
	attribute := func(name string) ([]float32, int, error) {
		accessorNumber, found := extension.Attributes[name]
		if !found {
			return nil, 0, nil
		}
		return file.readAccessor(accessorNumber)
	}
	translations, _, err := attribute("TRANSLATION")
	if err != nil {
		return nil, err
	}
	rotations, _, err := attribute("ROTATION")
	if err != nil {
		return nil, err
	}
	scales, _, err := attribute("SCALE")
	if err != nil {
		return nil, err
	}
	colors, colorComponents, err := attribute("_COLOR")
	if err != nil {
		return nil, err
	}
	count := max(len(translations)/3, len(rotations)/4, len(scales)/3)
	return &scene.Instances{
		Transforms: instanceTransforms(count, translations, rotations, scales),
		Colors:     instanceColors(count, colors, colorComponents),
	}, nil
}

func instanceTransforms(count int, translations, rotations, scales []float32) []graphics.Mat4 {
	transforms := make([]graphics.Mat4, count)
	for instance := 0; instance < count; instance++ {
		transforms[instance] = graphics.FromTranslationRotationScale(
			elementOrDefault(translations, instance, 3, graphics.Vec4{}),
			elementOrDefault(rotations, instance, 4, graphics.Vec4{0, 0, 0, 1}),
			elementOrDefault(scales, instance, 3, graphics.Vec4{1, 1, 1, 1}),
		)
	}
	return transforms
}

func instanceColors(count int, colors []float32, components int) []graphics.Vec4 {
	if components < 3 || len(colors) < count*components {
		return nil
	}
	result := make([]graphics.Vec4, count)
	for instance := 0; instance < count; instance++ {
		result[instance] = elementOrDefault(colors, instance, components, graphics.Vec4{1, 1, 1, 1})
	}
	return result
}

// elementOrDefault returns element number `index` of a flat array with `size` components per element.
func elementOrDefault(values []float32, index, size int, fallback graphics.Vec4) graphics.Vec4 {
	if (index+1)*size > len(values) {
		return fallback
	}
	return vectorOrDefault(values[index*size:(index+1)*size], fallback)
}

func vectorOrDefault(values []float32, fallback graphics.Vec4) graphics.Vec4 {
	result := fallback
	for component := 0; component < len(values) && component < 4; component++ {
		result[component] = values[component]
	}
	return result
}

func extraVector(extras map[string]any, key string, fallback graphics.Vec4) graphics.Vec4 {
	values, found := extras[key].([]any)
	if !found {
		return fallback
	}
	result := fallback
	for component := 0; component < len(values) && component < 4; component++ {
		if number, isNumber := values[component].(float64); isNumber {
			result[component] = float32(number)
		}
	}
	return result
}

func extraNumber(extras map[string]any, key string, fallback float32) float32 {
	if number, isNumber := extras[key].(float64); isNumber {
		return float32(number)
	}
	return fallback
}

// applyEnvironment overrides the environment with the values present in the scene extras.
func applyEnvironment(environment *scene.Environment, extras map[string]any) {
	environment.SunDirection = extraVector(extras, "sunDirection", environment.SunDirection).Normalize()
	environment.SunColor = extraVector(extras, "sunColor", environment.SunColor)
	environment.SkyZenith = extraVector(extras, "skyZenith", environment.SkyZenith)
	environment.SkyHorizon = extraVector(extras, "skyHorizon", environment.SkyHorizon)
	environment.GroundAmbient = extraVector(extras, "groundAmbient", environment.GroundAmbient)
	environment.Ambient = extraNumber(extras, "ambient", environment.Ambient)
	environment.VerticalScale = extraNumber(extras, "verticalScale", environment.VerticalScale)
	environment.FogColor = environment.SkyHorizon
	if fog, found := extras["fog"].(map[string]any); found {
		applyFog(environment, fog)
	}
	if water, found := extras["water"].(map[string]any); found {
		environment.Water = waterFromExtras(water)
	}
	if wind, found := extras["wind"].(map[string]any); found {
		environment.Wind.Direction = extraVector(wind, "direction", environment.Wind.Direction).Normalize()
		environment.Wind.Strength = extraNumber(wind, "strength", environment.Wind.Strength)
	}
}

func applyFog(environment *scene.Environment, fog map[string]any) {
	environment.FogColor = extraVector(fog, "color", environment.FogColor)
	environment.FogNear = extraNumber(fog, "near", 0)
	environment.FogFar = extraNumber(fog, "far", 0)
}

func waterFromExtras(water map[string]any) *scene.Water {
	return &scene.Water{
		Level:      extraNumber(water, "level", 0),
		Deep:       extraVector(water, "deep", graphics.Vec4{0.05, 0.22, 0.28, 0.85}),
		Shallow:    extraVector(water, "shallow", graphics.Vec4{0.3, 0.62, 0.6, 0.5}),
		Size:       extraNumber(water, "size", 1000),
		WaveLength: extraNumber(water, "waveLength", 0),
		FloorDepth: extraNumber(water, "floorDepth", 0),
		ColorDepth: extraNumber(water, "colorDepth", 0),
	}
}
