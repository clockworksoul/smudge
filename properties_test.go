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

package smudge

import (
	"testing"
)

func TestSplitString0a(t *testing.T) {
	str := ""
	split := splitDelimmitedString(str, stringListDelimitRegex)

	if len(split) != 0 {
		t.Errorf("len=%d contents=%v\n", len(split), split)
	}
}

func TestSplitString0b(t *testing.T) {
	str := " "
	split := splitDelimmitedString(str, stringListDelimitRegex)

	if len(split) != 0 {
		t.Errorf("len=%d contents=%v\n", len(split), split)
	}
}

func TestSplitString1(t *testing.T) {
	str := "foo"
	split := splitDelimmitedString(str, stringListDelimitRegex)

	if len(split) != 1 || split[0] != "foo" {
		t.Errorf("len=%d contents=%v\n", len(split), split)
	}
}

func TestSplitString2a(t *testing.T) {
	str := "foo bar"
	split := splitDelimmitedString(str, stringListDelimitRegex)

	if len(split) != 2 || split[0] != "foo" {
		t.Errorf("len=%d contents=%v\n", len(split), split)
	}
}

func TestSplitString2b(t *testing.T) {
	str := "foo, bar"
	split := splitDelimmitedString(str, stringListDelimitRegex)

	if len(split) != 2 || split[0] != "foo" {
		t.Errorf("len=%d contents=%v\n", len(split), split)
	}
}

func TestSplitString2c(t *testing.T) {
	str := "foo  bar"
	split := splitDelimmitedString(str, stringListDelimitRegex)

	if len(split) != 2 || split[0] != "foo" || split[1] != "bar" {
		t.Errorf("len=%d contents=%v\n", len(split), split)
	}
}

func TestSplitString3a(t *testing.T) {
	str := "foo bar bat"
	split := splitDelimmitedString(str, stringListDelimitRegex)

	if len(split) != 3 || split[0] != "foo" {
		t.Errorf("len=%d contents=%v\n", len(split), split)
	}
}

func TestSplitString3b(t *testing.T) {
	str := "foo, bar, bat"
	split := splitDelimmitedString(str, stringListDelimitRegex)

	if len(split) != 3 || split[0] != "foo" {
		t.Errorf("len=%d contents=%v\n", len(split), split)
	}
}

func TestSplitString3c(t *testing.T) {
	str := "foo bar, bat"
	split := splitDelimmitedString(str, stringListDelimitRegex)

	if len(split) != 3 || split[0] != "foo" {
		t.Errorf("len=%d contents=%v\n", len(split), split)
	}
}
