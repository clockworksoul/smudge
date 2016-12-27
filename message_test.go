/*
Copyright 2015 The Smudge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package smudge

import (
	"testing"
)

func TestEncodeDecodeUint16A(t *testing.T) {
	bytes := make([]byte, 2, 2)
	initial := uint16(0x0000)

	encodeUint16(initial, bytes, 0)

	backAgain := decodeUint16(bytes, 0)

	if len(bytes) != 2 {
		t.Fail()
	}

	if initial != backAgain {
		t.Errorf("%d != %d", initial, backAgain)
	}
}

func TestEncodeDecodeUint16B(t *testing.T) {
	bytes := make([]byte, 2, 2)
	initial := uint16(0x048C)

	encodeUint16(initial, bytes, 0)

	backAgain := decodeUint16(bytes, 0)

	if len(bytes) != 2 {
		t.Fail()
	}

	if initial != backAgain {
		t.Errorf("%d != %d", initial, backAgain)
	}
}

func TestEncodeDecodeUint16C(t *testing.T) {
	bytes := make([]byte, 2, 2)
	initial := uint16(0x159D)

	encodeUint16(initial, bytes, 0)

	backAgain := decodeUint16(bytes, 0)

	if len(bytes) != 2 {
		t.Fail()
	}

	if initial != backAgain {
		t.Errorf("%d != %d", initial, backAgain)
	}
}

func TestEncodeDecodeUint16D(t *testing.T) {
	bytes := make([]byte, 2, 2)
	initial := uint16(0xFFFF)

	encodeUint16(initial, bytes, 0)

	backAgain := decodeUint16(bytes, 0)

	if len(bytes) != 2 {
		t.Fail()
	}

	if initial != backAgain {
		t.Errorf("%d != %d", initial, backAgain)
	}
}

func TestEncodeDecodeUint32A(t *testing.T) {
	bytes := make([]byte, 4, 4)
	initial := uint32(0x00000000)

	encodeUint32(initial, bytes, 0)

	backAgain := decodeUint32(bytes, 0)

	if len(bytes) != 4 {
		t.Fail()
	}

	if initial != backAgain {
		t.Errorf("%d != %d", initial, backAgain)
	}
}

func TestEncodeDecodeUint32B(t *testing.T) {
	bytes := make([]byte, 4, 4)
	initial := uint32(0x02468ACE)

	encodeUint32(initial, bytes, 0)

	backAgain := decodeUint32(bytes, 0)

	if len(bytes) != 4 {
		t.Fail()
	}

	if initial != backAgain {
		t.Errorf("%d != %d", initial, backAgain)
	}
}

func TestEncodeDecodeUint32C(t *testing.T) {
	bytes := make([]byte, 4, 4)
	initial := uint32(0x13579BDF)

	encodeUint32(initial, bytes, 0)

	backAgain := decodeUint32(bytes, 0)

	if len(bytes) != 4 {
		t.Fail()
	}

	if initial != backAgain {
		t.Errorf("%d != %d", initial, backAgain)
	}
}

func TestEncodeDecodeUint32D(t *testing.T) {
	bytes := make([]byte, 4, 4)
	initial := uint32(0xFFFFFFFF)

	encodeUint32(initial, bytes, 0)

	backAgain := decodeUint32(bytes, 0)

	if len(bytes) != 4 {
		t.Fail()
	}

	if initial != backAgain {
		t.Errorf("%d != %d", initial, backAgain)
	}
}

func TestEncodeDecodeUint64A(t *testing.T) {
	bytes := make([]byte, 8, 8)
	initial := uint64(0x000000000000000)

	encodeUint64(initial, bytes, 0)

	backAgain := decodeUint64(bytes, 0)

	if len(bytes) != 8 {
		t.Fail()
	}

	if initial != backAgain {
		t.Errorf("%d != %d", initial, backAgain)
	}
}

func TestEncodeDecodeUint64B(t *testing.T) {
	bytes := make([]byte, 8, 8)
	initial := uint64(0x02468ACE02468ACE)

	encodeUint64(initial, bytes, 0)

	backAgain := decodeUint64(bytes, 0)

	if len(bytes) != 8 {
		t.Fail()
	}

	if initial != backAgain {
		t.Errorf("%d != %d", initial, backAgain)
	}
}

func TestEncodeDecodeUint64C(t *testing.T) {
	bytes := make([]byte, 8, 8)
	initial := uint64(0x13579BDF13579BDF)

	encodeUint64(initial, bytes, 0)

	backAgain := decodeUint64(bytes, 0)

	if len(bytes) != 8 {
		t.Fail()
	}

	if initial != backAgain {
		t.Errorf("%d != %d", initial, backAgain)
	}
}

func TestEncodeDecodeUint64D(t *testing.T) {
	bytes := make([]byte, 8, 8)
	initial := uint64(0xFFFFFFFFFFFFFFFF)

	encodeUint64(initial, bytes, 0)

	backAgain := decodeUint64(bytes, 0)

	if len(bytes) != 8 {
		t.Fail()
	}

	if initial != backAgain {
		t.Errorf("%d != %d", initial, backAgain)
	}
}
