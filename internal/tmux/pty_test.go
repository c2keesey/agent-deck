//go:build !windows
// +build !windows

package tmux

import (
	"fmt"
	"testing"
)

func TestIndexDetachKey_RawByte(t *testing.T) {
	data := []byte{0x01, 0x02, 0x11, 0x03} // Ctrl+Q at index 2
	if idx := IndexDetachKey(data, 0x11); idx != 2 {
		t.Fatalf("IndexDetachKey raw byte = %d, want 2", idx)
	}
}

func TestIndexDetachKey_NotFound(t *testing.T) {
	data := []byte("hello world")
	if idx := IndexDetachKey(data, 0x11); idx != -1 {
		t.Fatalf("IndexDetachKey not found = %d, want -1", idx)
	}
}

func TestIndexDetachKey_XtermModifyOtherKeys(t *testing.T) {
	// Ctrl+Q = byte 17, keyCode = 17 + 96 = 113 ('q')
	seq := fmt.Sprintf("\x1b[27;5;%d~", 113)
	data := []byte("prefix" + seq + "suffix")
	idx := IndexDetachKey(data, 17)
	if idx != 6 { // len("prefix") = 6
		t.Fatalf("IndexDetachKey xterm = %d, want 6", idx)
	}
}

func TestIndexDetachKey_CSIu(t *testing.T) {
	// Ctrl+Q = byte 17, keyCode = 113
	seq := fmt.Sprintf("\x1b[%d;5u", 113)
	data := []byte(seq)
	if idx := IndexDetachKey(data, 17); idx != 0 {
		t.Fatalf("IndexDetachKey CSI u = %d, want 0", idx)
	}
}

func TestIndexDetachKey_MRUSwitchByte(t *testing.T) {
	// Ctrl+W = byte 23
	data := []byte{0x41, 0x17, 0x42} // 'A', Ctrl+W, 'B'
	if idx := IndexDetachKey(data, 23); idx != 1 {
		t.Fatalf("IndexDetachKey Ctrl+W = %d, want 1", idx)
	}
}

func TestIndexDetachKey_DistinguishesMRUFromDetach(t *testing.T) {
	// Data contains Ctrl+W (23) but not Ctrl+Q (17)
	data := []byte{0x17, 0x41}
	if idx := IndexDetachKey(data, 23); idx != 0 {
		t.Fatalf("Ctrl+W index = %d, want 0", idx)
	}
	if idx := IndexDetachKey(data, 17); idx != -1 {
		t.Fatalf("Ctrl+Q index = %d, want -1 (not present)", idx)
	}
}

func TestErrMRUSwitch(t *testing.T) {
	if ErrMRUSwitch == nil {
		t.Fatal("ErrMRUSwitch should not be nil")
	}
	if ErrMRUSwitch.Error() != "mru switch requested" {
		t.Fatalf("ErrMRUSwitch message = %q, want %q", ErrMRUSwitch.Error(), "mru switch requested")
	}
}
