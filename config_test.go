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
	"errors"
	"fmt"
	"github.com/matryer/is"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	testCases := []struct {
		name    string
		input   map[string]string
		want    Config
		wantErr error
	}{
		{
			name:    "missing token",
			input:   map[string]string{},
			want:    Config{},
			wantErr: fmt.Errorf("params [%v]: %w", Token, ErrRequiredParamMissing),
		},
		{
			name: "full config",
			input: map[string]string{
				Token:        "test-token",
				PollInterval: "123s",
			},
			want: Config{
				token:        "test-token",
				pollInterval: 123 * time.Second,
			},
			wantErr: nil,
		},
		{
			name: "poll interval shorter than a minute",
			input: map[string]string{
				Token:        "test-token",
				PollInterval: "23s",
			},
			want:    Config{},
			wantErr: errors.New("poll interval must not be shorter than a minute (provided: 23s)"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)

			parsed, err := ParseConfig(tc.input)
			if tc.wantErr != nil {
				is.Equal(tc.wantErr, err)
			} else {
				is.NoErr(err)
				is.Equal(tc.want, parsed)
			}
		})
	}
}
