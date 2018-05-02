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
import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
)

// Tags contains
type Tags struct {
	tags  map[string]string
	mutex *sync.Mutex
}

// NewTags create a new set of tags
func NewTags() Tags {
	return Tags{
		tags:  make(map[string]string),
		mutex: &sync.Mutex{},
	}
}

var (
	// ErrInvalidChars is returned when a tag contains invalid chars
	ErrInvalidChars = errors.New("invalid characters in tag")
)

// Exists checks if the tag already exists
func (t *Tags) Exists(name string) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	name = strings.ToLower(strings.TrimSpace(name))
	_, exists := t.tags[name]
	return exists
}

// SetTag adds or replaces a tag. ErrInvalidChars are returned if the tag name
// or value contains invalid characters
func (t *Tags) SetTag(name string, value string) error {
	name = strings.ToLower(strings.TrimSpace(name))
	if !isValidIdentifier(name) || !isValidIdentifier(value) {
		return ErrInvalidChars
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.tags[name] = value
	return nil
}

// GetTag returns the tag value
func (t *Tags) GetTag(name string) (string, bool) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	val, ok := t.tags[strings.ToLower(name)]
	return val, ok
}

// RemoveTag removes the tag from the collection
func (t *Tags) RemoveTag(name string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	delete(t.tags, strings.ToLower(name))
}

// TagJSON returns the tags formatted as a JSON struct
func (t *Tags) TagJSON() []byte {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	buf, _ := json.Marshal(t.tags)
	return buf
}

// Tags return a copy of the tags
func (t *Tags) Tags() map[string]string {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	ret := make(map[string]string)
	for k, v := range t.tags {
		ret[k] = v
	}
	return ret
}

// Equals checks if the tags are the same for both
func (t *Tags) Equals(other Tags) bool {
	return reflect.DeepEqual(t.tags, other.tags)
}

// NewTagsFromBuffer unmarshals a JSON struct into a Tags structure
func NewTagsFromBuffer(buf []byte) (*Tags, error) {
	t := NewTags()
	if buf == nil || len(buf) == 0 {
		return &t, nil
	}
	if err := json.Unmarshal(buf, &t.tags); err != nil {
		return nil, err
	}
	return &t, nil
}

// NewTagsFromMap builds a new tags structure from a map
func NewTagsFromMap(values map[string]string) (*Tags, error) {
	t := NewTags()
	for k, v := range values {
		if err := t.SetTag(k, v); err != nil {
			return nil, err
		}
	}
	return &t, nil
}

var r *regexp.Regexp

func init() {
	var err error
	r, err = regexp.Compile("^[A-Za-z0-:_\\-+@\\ ,.=]*$")
	if err != nil {
		panic(fmt.Sprintf("I can't compile the string regexp: %v", err))
	}
}

// Valid tag identifiers are [A-Za-z0-9:_-+@]
func isValidIdentifier(value string) bool {
	return r.Match([]byte(value))
}
