package protocol

//
//Copyright 2018 Telenor Digital AS
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
import (
	"testing"
)

// Basic encoder tests
type encoder interface {
	encode(buffer []byte, pos *int) error
}

func basicEncoderTests(t *testing.T, e encoder) {
	buffer := make([]byte, 10)
	pos := 12
	if err := e.encode(buffer, &pos); err == nil {
		t.Fatal("Expected error with (less than) zero length buffer")
	}
	if err := e.encode(nil, &pos); err == nil {
		t.Fatal("Expected error with nil buffer")
	}
	if err := e.encode(buffer, nil); err == nil {
		t.Fatal("Expected error with nil pointer")
	}
}
