package loaders

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
)

type glbHeader struct {
	Magic   uint32
	Version uint32
	Length  uint32
}

type glbChunkHeader struct {
	Length uint32
	Type   uint32
}

type gltfJSON struct {
	Accessors   []gltfAccessor   `json:"accessors"`
	BufferViews []gltfBufferView `json:"bufferViews"`
	Buffers     []gltfBuffer     `json:"buffers"`
	Meshes      []gltfMesh       `json:"meshes"`
}

type gltfAccessor struct {
	BufferView    int    `json:"bufferView"`
	ComponentType int    `json:"componentType"`
	Count         int    `json:"count"`
	Type          string `json:"type"`
	ByteOffset    int    `json:"byteOffset"`
}

type gltfBufferView struct {
	Buffer     int `json:"buffer"`
	ByteOffset int `json:"byteOffset"`
	ByteLength int `json:"byteLength"`
}

type gltfBuffer struct {
	ByteLength int `json:"byteLength"`
}

type gltfMesh struct {
	Primitives []gltfPrimitive `json:"primitives"`
}

type gltfPrimitive struct {
	Attributes map[string]int `json:"attributes"`
	Indices    *int           `json:"indices"`
}

func LoadGLB(path string) (*models.Model, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var header glbHeader
	if err := binary.Read(file, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	if header.Magic != 0x46546C67 {
		return nil, fmt.Errorf("invalid GLB magic")
	}

	var jsonChunk gltfJSON
	var binaryChunk []byte

	for {
		var chunkHeader glbChunkHeader
		if err := binary.Read(file, binary.LittleEndian, &chunkHeader); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		chunkData := make([]byte, chunkHeader.Length)
		if _, err := io.ReadFull(file, chunkData); err != nil {
			return nil, err
		}

		if chunkHeader.Type == 0x4E4F534A { // JSON
			if err := json.Unmarshal(chunkData, &jsonChunk); err != nil {
				return nil, err
			}
		} else if chunkHeader.Type == 0x004E4942 { // BIN
			binaryChunk = chunkData
		}
	}

	model := &models.Model{}
	fmt.Printf("GLB: Found %d meshes\n", len(jsonChunk.Meshes))

	minX, minY, minZ := float32(math.MaxFloat32), float32(math.MaxFloat32), float32(math.MaxFloat32)
	maxX, maxY, maxZ := float32(-math.MaxFloat32), float32(-math.MaxFloat32), float32(-math.MaxFloat32)

	for mIdx, gMesh := range jsonChunk.Meshes {
		for pIdx, primitive := range gMesh.Primitives {
			mesh := models.Mesh{}

			// Load Positions
			posIdx, ok := primitive.Attributes["POSITION"]
			if !ok {
				continue
			}

			posAccessor := jsonChunk.Accessors[posIdx]
			fmt.Printf("Mesh %d Primitive %d: %d vertices\n", mIdx, pIdx, posAccessor.Count)
			posView := jsonChunk.BufferViews[posAccessor.BufferView]
			posOffset := posView.ByteOffset + posAccessor.ByteOffset
			
			// Assuming float32 VEC3 for simplicity
			for i := 0; i < posAccessor.Count; i++ {
				offset := posOffset + i*12
				ux := binary.LittleEndian.Uint32(binaryChunk[offset : offset+4])
				uy := binary.LittleEndian.Uint32(binaryChunk[offset+4 : offset+8])
				uz := binary.LittleEndian.Uint32(binaryChunk[offset+8 : offset+12])
				
				x := math.Float32frombits(ux)
				y := math.Float32frombits(uy)
				z := math.Float32frombits(uz)

				if x < minX { minX = x }
				if y < minY { minY = y }
				if z < minZ { minZ = z }
				if x > maxX { maxX = x }
				if y > maxY { maxY = y }
				if z > maxZ { maxZ = z }

				mesh.Vertices = append(mesh.Vertices, graphics.Vec4{x, y, z, 1.0})
			}

			// Load Indices
			if primitive.Indices != nil {
				indAccessor := jsonChunk.Accessors[*primitive.Indices]
				fmt.Printf("Mesh %d Primitive %d: %d indices\n", mIdx, pIdx, indAccessor.Count)
				indView := jsonChunk.BufferViews[indAccessor.BufferView]
				indOffset := indView.ByteOffset + indAccessor.ByteOffset

				for i := 0; i < indAccessor.Count; i++ {
					if indAccessor.ComponentType == 5123 { // UNSIGNED_SHORT
						offset := indOffset + i*2
						idx := binary.LittleEndian.Uint16(binaryChunk[offset : offset+2])
						mesh.Indices = append(mesh.Indices, uint32(idx))
					} else if indAccessor.ComponentType == 5125 { // UNSIGNED_INT
						offset := indOffset + i*4
						idx := binary.LittleEndian.Uint32(binaryChunk[offset : offset+4])
						mesh.Indices = append(mesh.Indices, idx)
					}
				}
			}

			model.Meshes = append(model.Meshes, mesh)
		}
	}

	fmt.Printf("GLB Bounding Box: Min(%f, %f, %f) Max(%f, %f, %f)\n", minX, minY, minZ, maxX, maxY, maxZ)

	return model, nil
}
