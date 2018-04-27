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
	"sort"

	"github.com/ExploratoryEngineering/logging"
)

// MACCommandSet is a collection of MAC commands. The command sets have an
// expected length when decoding (the packets hold the number of bytes) and
// an absolute limit when encoding.
type MACCommandSet struct {
	commands  map[CID]MACCommand
	maxLength int
	message   MType
}

// NewFOptsSet returns a set for the FOpts field. The max length is set to 16.
func NewFOptsSet(message MType) MACCommandSet {
	return NewMACCommandSet(message, MaxFOptsLen)
}

// NewMACCommandSet creates a new MACCommandSet instance. The message type
// indicates what kind of MAC commands we can expect and the maxLength parameter
// describes the expected length (when decoding a byte buffer) or the maximum
// length (when encoding into a buffer)
func NewMACCommandSet(message MType, maxLength int) MACCommandSet {
	return MACCommandSet{make(map[CID]MACCommand), maxLength, message}
}

// Add adds a new MAC command to the set
func (m *MACCommandSet) Add(cmd MACCommand) bool {
	if m.EncodedLength()+cmd.Length() > m.maxLength {
		logging.Error("Unable to add command to set. Current length is %d and the new length would be %d (%d is max)", m.EncodedLength(), m.EncodedLength()+cmd.Length(), m.maxLength)
		return false
	}
	if cmd.Uplink() != m.message.Uplink() {
		logging.Error("The command %T isn't the right type. Expected Uplink flag to be %v, not %v", cmd, m.message.Uplink(), cmd.Uplink())
		return false
	}
	m.commands[cmd.ID()] = cmd
	return true

}

// Contains returns true if the command set contains a command with the given CID
func (m *MACCommandSet) Contains(cid CID) bool {
	_, exists := m.commands[cid]
	return exists
}

// Remove removes a command from the set
func (m *MACCommandSet) Remove(cid CID) {
	delete(m.commands, cid)
}

// Clear removes all of the MAC commands in the set
func (m *MACCommandSet) Clear() {
	m.commands = make(map[CID]MACCommand)
}

// List returns a list of sorted MAC Commands
func (m *MACCommandSet) List() []MACCommand {
	ret := make([]MACCommand, 0)
	for _, v := range m.commands {
		ret = append(ret, v)
	}
	sort.Slice(ret, func(x, y int) bool {
		return ret[x].ID() < ret[y].ID()
	})
	return ret
}

// EncodedLength returns the current encoded length of the set
func (m *MACCommandSet) EncodedLength() int {
	currentLength := 0
	for _, v := range m.commands {
		currentLength += v.Length()
	}
	return currentLength
}

// Size returns the number of commands in the set
func (m *MACCommandSet) Size() int {
	return len(m.commands)
}

// Message returns true if the set contains uplink commands
func (m *MACCommandSet) Message() MType {
	return m.message
}

// Encode into a byte buffer
func (m *MACCommandSet) encode(buffer []byte, pos *int) error {
	for _, v := range m.List() {
		if err := v.encode(buffer, pos); err != nil {
			return err
		}
	}
	return nil
}

// Decode from byte buffer. This clears the contents.
func (m *MACCommandSet) decode(buffer []byte, pos *int) error {
	if pos == nil {
		return ErrNilError
	}
	if len(buffer) < *pos+1 {
		return ErrBufferTruncated
	}
	m.Clear()
	currentLength := 0
	for {
		if len(buffer) <= *pos {
			// end of buffer; no more commands
			return ErrBufferTruncated
		}
		cid := CID(buffer[*pos])
		var newCommand MACCommand
		if m.message.Uplink() {
			newCommand = NewUplinkMACCommand(cid)
		} else {
			newCommand = NewDownlinkMACCommand(cid)
		}
		// This is an unknown command. Stop reading
		if newCommand == nil {
			return errUnknownMAC
		}
		currentLength += newCommand.Length()
		if currentLength > m.maxLength {
			// Stop decoding. This set won't hold more commands
			return nil
		}
		if err := newCommand.decode(buffer, pos); err != nil {
			return err
		}
		// In theory the add command will always succeed since we are keeping
		// track of the overall length but... better be safe.
		if !m.Add(newCommand) {
			return ErrInvalidSource
		}
	}
}

// Copy will copy the contents of the other command set. If there's not enough
// room it will return false
func (m *MACCommandSet) Copy(other MACCommandSet) bool {
	for _, v := range other.commands {
		if !m.Add(v) {
			return false
		}
	}
	return true
}
