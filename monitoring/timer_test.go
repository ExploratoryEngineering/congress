package monitoring

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

func TestTimer(t *testing.T) {
	tk := NewTimer()

	tk.Begin(TimeEncoder)
	tk.End()

	tk.Begin(nil)

	tk.Begin(TimeDecoder)
	tk.Begin(nil)

	tk.Begin(TimeSchedulerProcess)
	tk.Begin(TimeSchedulerProcess)
	tk.End()

	tk.End()
}
