package message

import (
	"bytes"
	"github.com/google/go-cmp/cmp"
	"testing"
)

// Test Encode and Decode message Header
func TestHeader_Encode(t *testing.T) {
	header := Header{
		Version:      0,
		MessageType:  MsgTypeRequest,
		CompressType: NoneCompress,
		SeqID:        12345678,
		Extension:    0,
		BodyLength:   100100,
	}

	hrd := EncodeHeader(header)

	h, err := DecodeHeader(bytes.NewReader(hrd))
	if err != nil {
		t.Errorf("DecodeHeader(%v) Fail", h)
		return
	}

	diff := cmp.Diff(header, h)
	if diff != "" {
		t.Fatalf(diff)
	}
}
