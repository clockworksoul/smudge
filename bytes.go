/*
Copyright 2016 The Smudge Authors.

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

// This file contains a variety of methods to encode/decode various primitive
// data types into a byte slice using a fixed length encoding scheme. We
// chose fixed lengths (as opposed to the variable length encoding provided by
// the encoding/binary package) because it makes decoding easier. It's also
// (very slightly) more efficient.
//
// These functions are used mostly by the components defined in message.go.

package smudge

func decodeUint16(bytes []byte, startIndex int) (uint16, int) {
	var number uint16 = 0

	number = uint16(bytes[startIndex+1])<<8 |
		uint16(bytes[startIndex+0])

	return number, startIndex + 2
}

func decodeUint32(bytes []byte, startIndex int) (uint32, int) {
	var number uint32 = 0

	number = uint32(bytes[startIndex+3])<<24 |
		uint32(bytes[startIndex+2])<<16 |
		uint32(bytes[startIndex+1])<<8 |
		uint32(bytes[startIndex+0])

	return number, startIndex + 4
}

func decodeUint64(bytes []byte, startIndex int) (uint64, int) {
	var number uint64 = 0

	number = uint64(bytes[startIndex+7])<<56 |
		uint64(bytes[startIndex+6])<<48 |
		uint64(bytes[startIndex+5])<<40 |
		uint64(bytes[startIndex+4])<<32 |
		uint64(bytes[startIndex+3])<<24 |
		uint64(bytes[startIndex+2])<<16 |
		uint64(bytes[startIndex+1])<<8 |
		uint64(bytes[startIndex+0])

	return number, startIndex + 8
}

func encodeUint16(number uint16, bytes []byte, startIndex int) int {
	bytes[startIndex+0] = byte(number)
	bytes[startIndex+1] = byte(number >> 8)

	return 2
}

func encodeUint32(number uint32, bytes []byte, startIndex int) int {
	bytes[startIndex+0] = byte(number)
	bytes[startIndex+1] = byte(number >> 8)
	bytes[startIndex+2] = byte(number >> 16)
	bytes[startIndex+3] = byte(number >> 24)

	return 4
}

func encodeUint64(number uint64, bytes []byte, startIndex int) int {
	bytes[startIndex+0] = byte(number)
	bytes[startIndex+1] = byte(number >> 8)
	bytes[startIndex+2] = byte(number >> 16)
	bytes[startIndex+3] = byte(number >> 24)
	bytes[startIndex+4] = byte(number >> 32)
	bytes[startIndex+5] = byte(number >> 40)
	bytes[startIndex+6] = byte(number >> 48)
	bytes[startIndex+7] = byte(number >> 56)

	return 8
}
