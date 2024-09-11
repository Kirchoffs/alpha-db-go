package alphadbgo

import (
	"testing"
)

func TestBNodeHeaderEncodeAndDecode(t *testing.T) {
	node := make(BNode, NODE_HEADER)
	node.setHeader(NODE_INTERNAL, 0)
	if node.category() != NODE_INTERNAL {
		t.Errorf("Expected %d, got %d", NODE_INTERNAL, node.category())
	}
	if node.numKeys() != 0 {
		t.Errorf("Expected %d, got %d", 0, node.numKeys())
	}

	node.setHeader(NODE_LEAF, 1)
	if node.category() != NODE_LEAF {
		t.Errorf("Expected %d, got %d", NODE_LEAF, node.category())
	}
	if node.numKeys() != 1 {
		t.Errorf("Expected %d, got %d", 1, node.numKeys())
	}
}
