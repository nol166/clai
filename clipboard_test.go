package main

import (
	"bytes"
	"testing"
)

func TestEncodeForClip(t *testing.T) {
	// little-endian code units, no BOM (clip.exe keeps a BOM as text);
	// 😀 needs a surrogate pair
	want := []byte{
		'h', 0x00, 'i', 0x00,
		0x14, 0x20, // — (U+2014)
		0x3D, 0xD8, 0x00, 0xDE, // 😀 (U+1F600)
	}
	if got := encodeForClip("hi—😀"); !bytes.Equal(got, want) {
		t.Errorf("got % x, want % x", got, want)
	}
}
