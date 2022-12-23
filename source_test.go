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
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/matryer/is"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notion "github.com/conduitio-labs/notionapi"
	"github.com/conduitio-labs/notionapi/mock"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

func TestSource_Config_FailsWhenEmpty(t *testing.T) {
	is := is.New(t)
	underTest := NewSource()
	err := underTest.Configure(context.Background(), make(map[string]string))

	is.True(errors.Is(err, ErrRequiredParamMissing))
}

func TestSource_Teardown_NoOpen(t *testing.T) {
	is := is.New(t)
	underTest := NewSource()
	err := underTest.Teardown(context.Background())
	is.NoErr(err)
}

func TestSource_Open_NilPosition(t *testing.T) {
	is := is.New(t)
	underTest := NewSource().(*Source)
	err := underTest.Open(context.Background(), nil)
	is.NoErr(err)
	is.True(underTest.lastMinuteRead.IsZero())
}

func TestSource_Open_WithPosition(t *testing.T) {
	is := is.New(t)
	underTest := NewSource().(*Source)
	pos := position{
		ID:             "test-id",
		LastEditedTime: time.Now(),
	}
	sdkPos, err := pos.toSDKPosition()
	is.NoErr(err)

	err = underTest.Open(context.Background(), sdkPos)
	is.NoErr(err)
	is.True(pos.LastEditedTime.Equal(underTest.lastMinuteRead))
}

func TestSource_Configure(t *testing.T) {
	tests := []struct {
		desc   string
		input  map[string]string
		output Config
		err    error
	}{
		{
			desc: "Succeed without override",
			input: map[string]string{
				Token: "abc-def",
			},
			output: Config{token: "abc-def", pollInterval: time.Minute},
		},
		{
			desc: "Succeed with override",
			input: map[string]string{
				Token:        "abc-def",
				PollInterval: "2m",
			},
			output: Config{token: "abc-def", pollInterval: 2 * time.Minute},
		},
		{
			desc: "Fail to override poll interval because too small",
			input: map[string]string{
				Token:        "abc-def",
				PollInterval: "1s",
			},
			err: fmt.Errorf("poll interval must not be shorter than a minute (provided: 1s)"),
		},
		{
			desc: "Fail to override poll interval because not a time",
			input: map[string]string{
				Token:        "abc-def",
				PollInterval: "a",
			},
			err: fmt.Errorf("cannot parse poll interval \"a\": time: invalid duration \"a\""),
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.Background()
			s := &Source{}

			err := s.Configure(ctx, tc.input)
			if tc.err == nil {
				require.NoError(t, err)
				assert.Equal(t, tc.output.token, s.config.token)
				assert.Equal(t, tc.output.pollInterval, s.config.pollInterval)
			} else {
				assert.Equal(t, tc.err.Error(), err.Error())
			}
		})
	}
}

func TestSource_Read(t *testing.T) {
	tests := []struct {
		desc           string
		client         func(ctrl *gomock.Controller) *notion.Client
		expectedRecord sdk.Record
		err            error
	}{
		{
			desc: "Succeed with empty responses",
			client: func(ctrl *gomock.Controller) *notion.Client {
				c := mock.NewMockNotionClient(ctrl)
				/*
					c.Search.EXPECT().
						Do(mock.Any(), &notion.SearchRequest{
							StartCursor: notion.Cursor{},
							Sort: &notion.SortObject{
								Direction: notion.SortOrderASC,
								Timestamp: notion.TimestampLastEdited,
							},
							Filter: map[string]string{
								"property": "object",
								"value":    "page",
							},
						}).Return(&notion.SearchResponse{}, nil).Times(1)

				*/
				return c
			},
		},
	}
	for _, tc := range tests {
		ctrl := gomock.NewController(t)
		s := &Source{client: tc.client(ctrl)}

		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.Background()

			record, err := s.Read(ctx)
			if tc.err == nil {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedRecord, record)
			} else {
				assert.Equal(t, tc.err, err)
			}
		})
	}
}
