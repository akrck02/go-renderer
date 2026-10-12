package loaders

import (
	"testing"
)

func TestNodeExtrasBecomeTheOverlay(t *testing.T) {
	document := map[string]any{
		"asset":  map[string]any{"version": "2.0"},
		"scene":  0,
		"scenes": []any{map[string]any{"nodes": []int{0, 1, 2, 3}}},
		"nodes": []any{
			map[string]any{"name": "lighthouse", "translation": []float32{5, 0, 0},
				"extras": map[string]any{"label": "Lighthouse", "icon": "lighthouse", "badge": "on", "label_color": "#2563eb"}},
			map[string]any{"name": "clock", "extras": map[string]any{"label": "12:00", "icon": "clock", "pinned": 1}},
			map[string]any{"name": "tide", "extras": map[string]any{"label": "Tide rising", "pinned": 2}},
			map[string]any{"name": "rock"},
		},
	}
	loaded, err := LoadScene(writeGLB(t, document, nil))
	if err != nil {
		t.Fatal(err)
	}
	labels := loaded.Overlay.Labels
	if len(labels) != 1 || labels[0].Text != "Lighthouse" || labels[0].Icon != "lighthouse" || labels[0].Badge != "on" {
		t.Fatalf("one label for the lighthouse, got %+v", labels)
	}
	if labels[0].Node != loaded.Nodes[0] || labels[0].Tint[2] < 0.9 {
		t.Fatalf("anchored to its node and tinted blue: %+v", labels[0])
	}
	header := loaded.Overlay.Header
	if len(header) != 2 || header[0].Text != "12:00" || header[1].Text != "Tide rising" {
		t.Fatalf("pinned nodes go to the header in order, got %+v", header)
	}
}
