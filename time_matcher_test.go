// Copyright © 2022 Meroxa, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package notion

import (
	"fmt"
	is "github.com/matryer/is"
	"reflect"
	"testing"
	"time"
)

type zeroTimeMatcher struct {
}

func (z zeroTimeMatcher) Matches(x interface{}) bool {
	if x == nil {
		return false
	}
	val := reflect.ValueOf(x)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}
	other, ok := val.Interface().(time.Time)
	if !ok {
		return false
	}

	return other.IsZero()
}

func (z zeroTimeMatcher) String() string {
	return "is zero time"
}

type timeEqMatcher struct {
	t time.Time
}

func (t timeEqMatcher) Matches(x interface{}) bool {
	if x == nil {
		return false
	}
	val := reflect.ValueOf(x)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}
	other, ok := val.Interface().(time.Time)
	if !ok {
		return false
	}

	return t.t.Equal(other)
}

func (t timeEqMatcher) String() string {
	return fmt.Sprintf("time is equal to %v (%T)", t.t, t.t)
}

func TestTimeEqMatcher_Nil(t *testing.T) {
	is := is.New(t)
	matches := timeEqMatcher{time.Now()}.Matches(nil)
	is.True(!matches)
}

func TestTimeEqMatcher_Match(t *testing.T) {
	is := is.New(t)
	t1, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z07:00")
	t2, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z07:00")
	matches := timeEqMatcher{t1}.Matches(t2)
	is.True(matches)
}

func TestTimeEqMatcher_NoMatch(t *testing.T) {
	is := is.New(t)
	t1, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z07:00")
	matches := timeEqMatcher{t1}.Matches(time.Now())
	is.True(!matches)
}

func TestTimeEqMatcher_Match_Ptr(t *testing.T) {
	is := is.New(t)
	t1, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z07:00")
	t2, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z07:00")
	matches := timeEqMatcher{t1}.Matches(&t2)
	is.True(matches)
}

func TestTimeEqMatcher_NoMatch_Ptr(t *testing.T) {
	is := is.New(t)
	t1, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z07:00")
	t2 := time.Now()
	matches := timeEqMatcher{t1}.Matches(&t2)
	is.True(!matches)
}
