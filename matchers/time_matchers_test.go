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

package matchers

import (
	"testing"
	"time"

	"github.com/matryer/is"
)

func TestZeroTimeMatcher_Nil(t *testing.T) {
	is := is.New(t)
	matches := TimeIsZero.Matches(nil)
	is.True(!matches)
}

func TestZeroTimeMatcher_Match(t *testing.T) {
	is := is.New(t)
	matches := TimeIsZero.Matches(time.Time{})
	is.True(matches)
}

func TestZeroTimeMatcher_NoMatch(t *testing.T) {
	is := is.New(t)
	matches := TimeIsZero.Matches(time.Now())
	is.True(!matches)
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
