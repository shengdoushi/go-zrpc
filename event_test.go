/*
Copyright © Max Mazurov (fox.cpp) 2018

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package zrpc

import (
	"bytes"
	"strings"
	"testing"

	"gotest.tools/assert"
)

func TestEvent_MarshalUnmarshal(t *testing.T) {
	t.Run("example event (args=5)", func(t *testing.T) {
		ev := event{
			Hdr: eventHdr{
				Version:    3,
				MsgID:      "5a741c23675b4ae18c7441da24d1f9cf",
				ResponseTo: "",
			},
			Name: "event_name_goes_here",
			Args: int64(5),
		}

		bin, err := ev.MarshalBinary()
		assert.NilError(t, err, "event.MarshalBinary failed")

		ev2 := event{}
		assert.NilError(t, ev2.UnmarshalBinary(bytes.NewReader(bin)), "event.UnmarshalBinary failed")
		assert.DeepEqual(t, ev, ev2)
	})
	t.Run("example event (args=[1,2,3], respTo=abc)", func(t *testing.T) {
		ev := event{
			Hdr: eventHdr{
				Version:    3,
				MsgID:      "5a741c25675b4ae18c7441da24d1f9cf",
				ResponseTo: "abc",
			},
			Name: "event_name_goes_here",
			Args: []interface{}{int64(1), int64(2), int64(3)},
		}

		bin, err := ev.MarshalBinary()
		assert.NilError(t, err, "event.MarshalBinary failed")

		ev2 := event{}
		assert.NilError(t, ev2.UnmarshalBinary(bytes.NewReader(bin)), "event.UnmarshalBinary failed")
		assert.DeepEqual(t, ev, ev2)
	})
	// TODO: Can we use fuzzy-testing here? (generate set of random events and pass them through marshal-unmarshal).
}

func checkEventParseFail(t *testing.T, encoded string) {
	ev := event{}
	err := ev.UnmarshalBinary(strings.NewReader(encoded))
	if err == nil {
		t.Fatalf("Successfully 'parsed' an invalid event:\n%+v\n", ev)
	}
}

func TestEvent_UnmarshalBinary(t *testing.T) {
	// Test for on-wire compatibility with reference implementation.
	t.Run("example event (args=5)", func(t *testing.T) {
		evStr := "\x93\x82\xaamessage_id\xc4 5a741c23675b4ae18c7441da24d1f9cf\xa1v\x03\xb4event_name_goes_here\x05"
		ev := event{}
		assert.NilError(t, ev.UnmarshalBinary(strings.NewReader(evStr)))

		assert.Equal(t, ev.Hdr.Version, 3, "protocol version mismatch")
		assert.Equal(t, ev.Hdr.MsgID, "5a741c23675b4ae18c7441da24d1f9cf", "msgid mismatch")
		assert.Equal(t, ev.Hdr.ResponseTo, "", "response_to present but shouldn't")

		assert.Equal(t, ev.Name, "event_name_goes_here")
		assert.Equal(t, ev.Args, int64(5))
	})
	t.Run("example event (args=[1,2,3], respTo=abc)", func(t *testing.T) {
		// Event encoding generated by reference implementation (zerorpc-python).
		evStr := "\x93\x83\xaamessage_id\xc4 5a741c25675b4ae18c7441da24d1f9cf\xa1v\x03\xabresponse_to\xa3abc\xb4event_name_goes_here\x93\x01\x02\x03"
		ev := event{}
		assert.NilError(t, ev.UnmarshalBinary(strings.NewReader(evStr)), "event.UnmarhshalBinary failed")
		assert.Equal(t, ev.Hdr.Version, 3, "protocol version mismatch")
		assert.Equal(t, ev.Hdr.MsgID, "5a741c25675b4ae18c7441da24d1f9cf", "msgid mismatch")
		assert.Equal(t, ev.Hdr.ResponseTo, "abc", "response_to mismatch")

		assert.Equal(t, ev.Name, "event_name_goes_here")
		assert.DeepEqual(t, ev.Args, []interface{}{int64(1), int64(2), int64(3)})
	})
	t.Run("malformed event (invalid msgpack)", func(t *testing.T) {
		t.Run("invalid msgpack", func(t *testing.T) {
			checkEventParseFail(t, "\x93abcdef")
		})
		t.Run("valid but irrelevant msgpack", func(t *testing.T) {
			checkEventParseFail(t, "\x83\x01\x80\x02\xa6abcdef\x03\xc0")
		})
		t.Run("non-string name", func(t *testing.T) {
			checkEventParseFail(t, "\x93\x82\xaamessage_id\xa3abc\xa1v\x03\xcc\xea\xc0")
		})
		t.Run("invalid version", func(t *testing.T) {
			checkEventParseFail(t, "\x93\x82\xaamessage_id\xa3abc\xa1v\xa3abc\xa6abcdef\xc0")
		})
		t.Run("no version", func(t *testing.T) {
			checkEventParseFail(t, "\x93\x81\xaamessage_id\xa3abc\xa6abcdef\xc0")
		})
		t.Run("no msgid", func(t *testing.T) {
			checkEventParseFail(t, "\x93\x80\xa6abcdef\xc0")
		})
	})
}
