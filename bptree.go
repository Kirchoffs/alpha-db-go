package alphadbgo

import (
	"bytes"
	"encoding/binary"
)

const NODE_PAGE_SIZE = 4096
const NODE_CATEGORY = 2
const NODE_NUM_KEYS = 2
const NODE_HEADER = NODE_CATEGORY + NODE_NUM_KEYS
const NODE_POINTER = 8
const NODE_OFFSET = 2
const NODE_KEY_LENGTH = 2
const NODE_VALUE_LENGTH = 2
const NODE_MAX_KEY = 1000
const NODE_MAX_VALUE = 3000

// category(2B) | num_keys(2B) | pointers(8B * num_keys) | offset(2B * num_keys) | key_length(2B) | value_length(2B) | key(1000B) | value(3000B)
func init() {
	maxNodeSpace := NODE_HEADER + NODE_POINTER + NODE_OFFSET + NODE_KEY_LENGTH + NODE_VALUE_LENGTH + NODE_MAX_KEY + NODE_MAX_VALUE
	assert(maxNodeSpace < NODE_PAGE_SIZE, "Node size is too large")
}

type BNode []byte

type BTree struct {
	root uint64
	get  func(uint64) []byte
	new  func([]byte) uint64
	del  func(uint64)
}

const (
	NODE_INTERNAL = 1
	NODE_LEAF     = 2
)

func (node BNode) category() uint16 {
	return binary.LittleEndian.Uint16(node[:2])
}

func (node BNode) numKeys() uint16 {
	return binary.LittleEndian.Uint16(node[2:])
}

func (node BNode) setHeader(category uint16, numKeys uint16) {
	binary.LittleEndian.PutUint16(node[0:], category)
	binary.LittleEndian.PutUint16(node[2:], numKeys)
}

func (node BNode) getChildPtr(idx uint16) uint64 {
	assert(idx < node.numKeys(), "Index out of range")
	pos := NODE_HEADER + idx*NODE_POINTER
	return binary.LittleEndian.Uint64(node[pos:])
}

func (node BNode) setChildPtr(idx uint16, ptr uint64) {
	assert(idx < node.numKeys(), "Index out of range")
	pos := NODE_HEADER + idx*NODE_POINTER
	binary.LittleEndian.PutUint64(node[pos:], ptr)
}

func offsetPos(node BNode, idx uint16) uint16 {
	assert(idx < node.numKeys(), "Index out of range")
	return NODE_HEADER + NODE_POINTER*node.numKeys() + idx*NODE_OFFSET
}

func (node BNode) getOffset(idx uint16) uint16 {
	return binary.LittleEndian.Uint16(node[offsetPos(node, idx):])
}

func (node BNode) setOffset(idx uint16, offset uint16) {
	binary.LittleEndian.PutUint16(node[offsetPos(node, idx):], offset)
}

func (node BNode) kvPos(idx uint16) uint16 {
	assert(idx <= node.numKeys(), "Index out of range")
	return NODE_HEADER + NODE_POINTER*node.numKeys() + NODE_OFFSET*node.numKeys() + node.getOffset(idx)
}

func (node BNode) getKey(idx uint16) []byte {
	assert(idx < node.numKeys(), "Index out of range")

	pos := node.kvPos(idx)
	klen := binary.LittleEndian.Uint16(node[pos:])
	return node[pos+NODE_KEY_LENGTH:][:klen]
}

func (node BNode) getVal(idx uint16) []byte {
	assert(idx < node.numKeys(), "Index out of range")

	pos := node.kvPos(idx)
	klen := binary.LittleEndian.Uint16(node[pos:])
	vlen := binary.LittleEndian.Uint16(node[pos+NODE_KEY_LENGTH+klen:])
	return node[pos+NODE_KEY_LENGTH+klen+NODE_VALUE_LENGTH:][:vlen]
}

func (node BNode) numBytes() uint16 {
	return node.kvPos(node.numKeys())
}

func nodeLookupLE(node BNode, key []byte) (uint16, bool) {
	numKeys := node.numKeys()
	l, r := uint16(0), numKeys
	for l < r {
		m := l + (r-l)/2
		if bytes.Compare(node.getKey(m), key) <= 0 {
			l = m + 1
		} else {
			r = m
		}
	}

	return l - 1, l != 0
}

func leafInsert(new BNode, old BNode, idx uint16, key []byte, val []byte) {
	new.setHeader(NODE_LEAF, old.numKeys()+1)
	nodeAppendRange(new, old, 0, 0, idx)
	nodeAppendKV(new, idx, 0, key, val)
	nodeAppendRange(new, old, idx+1, idx, old.numKeys()-idx)
}

func nodeAppendKV(new BNode, idx uint16, ptr uint64, key []byte, val []byte) {
	new.setChildPtr(idx, ptr)
	pos := new.kvPos(idx)
	binary.LittleEndian.PutUint16(new[pos:], uint16(len(key)))
	binary.LittleEndian.PutUint16(new[pos+NODE_KEY_LENGTH:], uint16(len(val)))
	copy(new[pos+NODE_KEY_LENGTH+NODE_VALUE_LENGTH:], key)
	copy(new[pos+NODE_KEY_LENGTH+NODE_VALUE_LENGTH+uint16(len(key)):], val)
	new.setOffset(idx+1, new.getOffset(idx)+NODE_KEY_LENGTH+NODE_VALUE_LENGTH+uint16(len(key)+len(val)))
}

func nodeAppendRange(new BNode, old BNode, dstNew uint16, srcOld uint16, n uint16) {
	for i := uint16(0); i < n; i++ {
		new.setChildPtr(dstNew+i, old.getChildPtr(srcOld+i))
		oldPos := old.kvPos(srcOld + i)
		klen := binary.LittleEndian.Uint16(old[oldPos:])
		vlen := binary.LittleEndian.Uint16(old[oldPos+NODE_KEY_LENGTH+klen:])
		k := old[oldPos+NODE_KEY_LENGTH+NODE_VALUE_LENGTH:][:klen]
		v := old[oldPos+NODE_KEY_LENGTH+NODE_VALUE_LENGTH+klen:][:vlen]
		newOffset := calculateNewOffset(new, dstNew+i)
		new.setOffset(dstNew+i, newOffset)

		binary.LittleEndian.PutUint16(new[new.kvPos(dstNew+i):], klen)
		binary.LittleEndian.PutUint16(new[new.kvPos(dstNew+i)+NODE_KEY_LENGTH:], vlen)
		copy(new[new.kvPos(dstNew+i):], k)
		copy(new[new.kvPos(dstNew+i)+NODE_KEY_LENGTH:], v)
	}
}

func calculateNewOffset(new BNode, idx uint16) uint16 {
	if idx == 0 {
		return 0
	}

	preKLen := binary.LittleEndian.Uint16(new[new.kvPos(idx-1):])
	preVLen := binary.LittleEndian.Uint16(new[new.kvPos(idx-1)+NODE_KEY_LENGTH:])
	return new.getOffset(idx-1) + NODE_KEY_LENGTH + NODE_VALUE_LENGTH + preKLen + preVLen
}

func nodeReplaceKidN(tree *BTree, new BNode, old BNode, idx uint16, kids ...BNode) {
	inc := uint16(len(kids))
	new.setHeader(old.category(), old.numKeys()+inc-1)
	nodeAppendRange(new, old, 0, 0, idx)
	for i, node := range kids {
		nodeAppendKV(new, idx+uint16(i), tree.new(node), node.getKey(0), nil)
	}
	nodeAppendRange(new, old, idx+inc, idx+1, old.numKeys()-(idx+1))
}

func nodeSplit2(left BNode, right BNode, old BNode) {

}

func nodeSplit3(old BNode) (uint16, [3]BNode) {

}
