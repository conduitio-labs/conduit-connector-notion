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

package test

import (
	"fmt"
	"reflect"
	"time"
)

var TimeIsZero = zeroTimeMatcher{}

type zeroTimeMatcher struct {
}

func (z zeroTimeMatcher) Matches(x interface{}) bool {
	if x == nil {
		return false
	}
	val := reflect.ValueOf(x)
	if val.Kind() == reflect.Pointer {
		return false
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

func TimeEq(t time.Time) timeEqMatcher {
	return timeEqMatcher{t: t}
}

func (t timeEqMatcher) Matches(x interface{}) bool {
	if x == nil {
		return false
	}
	val := reflect.ValueOf(x)
	if val.Kind() == reflect.Pointer {
		return false
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
