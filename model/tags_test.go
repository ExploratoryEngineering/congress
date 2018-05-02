package model

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
import "testing"

func TestTags(t *testing.T) {
	tags := NewTags()

	tags.SetTag("Hello", "World")
	val, ok := tags.GetTag("Hello")
	if !ok {
		t.Fatal("Couldn't find attribute")
	}
	if val != "World" {
		t.Fatal("Didn't get the expected attribute")
	}

	if !tags.Exists("Hello") || !tags.Exists("hello") || !tags.Exists("   heLLo  ") {
		t.Fatal("Hello should exist")
	}

	if err := tags.SetTag("Hello", "There"); err != nil {
		t.Fatal("Should be able to overwrite tag")
	}
	if err := tags.SetTag("Hello()", "Invalid()"); err == nil {
		t.Fatal("Should not be allowed to use invalid chars")
	}

	_, ok = tags.GetTag("Nothing")
	if ok {
		t.Fatal("Got value but did not expect one")
	}

	buf := tags.TagJSON()
	tags2, err := NewTagsFromBuffer(buf)
	if err != nil {
		t.Fatal("Couldn't read from buffer")
	}

	val, ok = tags2.GetTag("Hello")
	if !ok {
		t.Fatal("Attribute doesn't exist")
	}

	_, err = NewTagsFromBuffer(nil)
	if err != nil {
		t.Fatal("Didn't expect errors here")
	}

	vals := tags2.Tags()
	if _, ok := vals["hello"]; !ok {
		t.Fatal("Missing key/value")
	}

	tags.RemoveTag("hello")
	tags.RemoveTag("hello")
	tags.RemoveTag("bonjour")

	// Tags should be empty
	tags3 := tags.Tags()
	if len(tags3) > 0 {
		t.Fatal("There's still tags in the collection")
	}
}

func TestInvalidChars(t *testing.T) {
	if !isValidIdentifier("name") {
		t.Fatal("Name is valid, right?")
	}
	if !isValidIdentifier("Name name name") {
		t.Fatal("Name name name should work")
	}
	if !isValidIdentifier("Name-_@+: foo") {
		t.Fatal("should work")
	}
	if isValidIdentifier("Name-_@+: foo()") {
		t.Fatal("shouldn't work")
	}
}

func TestTagsFromMap(t *testing.T) {
	vals := map[string]string{
		"Foo":    "Bar",
		"Baz":    "Foo",
		"FooBar": "BarBaz",
	}

	tags, err := NewTagsFromMap(vals)
	if err != nil {
		t.Fatal("Got error converting map to tags: ", err)
	}
	if _, exists := tags.GetTag("foo"); !exists {
		t.Fatal("Missing value foo")
	}
	if _, exists := tags.GetTag("baZ"); !exists {
		t.Fatal("Missing value baz")
	}
	if _, exists := tags.GetTag("foobar"); !exists {
		t.Fatal("Missing value foobar")
	}

	invalidVals := map[string]string{
		"some":                         "invalid<>",
		"[other]":                      "invalid",
		"<script>alert('1');</script>": "Invalid",
	}
	if _, err := NewTagsFromMap(invalidVals); err == nil {
		t.Fatal("Expected error with invalid values")
	}
}
