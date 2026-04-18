package util

import (
	"testing"
)

func TestMergeMap_NoKey(t *testing.T) {
	target := map[string]interface{}{"a": 1}
	source := map[string]interface{}{"b": 2, "c": 3}

	MergeMap(target, source, "")

	if target["a"] != 1 {
		t.Errorf("Expected 'a' to remain 1, got %v", target["a"])
	}
	if target["b"] != 2 {
		t.Errorf("Expected 'b' to be 2, got %v", target["b"])
	}
	if target["c"] != 3 {
		t.Errorf("Expected 'c' to be 3, got %v", target["c"])
	}
}

func TestMergeMap_WithKey(t *testing.T) {
	target := map[string]interface{}{"a": 1}
	source := map[string]interface{}{"b": 2, "c": 3}

	MergeMap(target, source, "nested")

	nested, ok := target["nested"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'nested' to be a map")
	}
	if nested["b"] != 2 {
		t.Errorf("Expected nested['b'] to be 2, got %v", nested["b"])
	}
	if nested["c"] != 3 {
		t.Errorf("Expected nested['c'] to be 3, got %v", nested["c"])
	}
}

func TestMergeMap_EmptySource(t *testing.T) {
	target := map[string]interface{}{"a": 1}
	source := map[string]interface{}{}

	MergeMap(target, source, "nested")

	if len(target) != 1 {
		t.Errorf("Expected target to have 1 key, got %d", len(target))
	}
}

func TestGetOrCreateMap_Existing(t *testing.T) {
	target := map[string]interface{}{
		"nested": map[string]interface{}{"a": 1},
	}

	result := GetOrCreateMap(target, "nested")
	result["b"] = 2

	nested := target["nested"].(map[string]interface{})
	if nested["a"] != 1 || nested["b"] != 2 {
		t.Errorf("Expected nested map to have a=1, b=2, got %v", nested)
	}
}

func TestGetOrCreateMap_NonExistent(t *testing.T) {
	target := map[string]interface{}{"a": 1}

	result := GetOrCreateMap(target, "nested")
	result["b"] = 2

	nested := target["nested"].(map[string]interface{})
	if nested["b"] != 2 {
		t.Errorf("Expected nested['b'] to be 2, got %v", nested["b"])
	}
}

func TestGetOrCreateMap_NilTarget(t *testing.T) {
	var target map[string]interface{}
	result := GetOrCreateMap(target, "key")

	if result != nil {
		t.Errorf("Expected nil result for nil target, got %v", result)
	}
}

func TestGetOrCreateMap_NonMapValue(t *testing.T) {
	target := map[string]interface{}{"key": "string value"}

	result := GetOrCreateMap(target, "key")
	result["nested"] = 1

	nested := target["key"].(map[string]interface{})
	if nested["nested"] != 1 {
		t.Errorf("Expected nested['nested'] to be 1, got %v", nested["nested"])
	}
}
