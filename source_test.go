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
	"github.com/conduitio-labs/conduit-connector-notion/client"
	"github.com/conduitio-labs/conduit-connector-notion/mock"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"testing"
	"time"

	"github.com/matryer/is"
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

func TestSource_Read_NoPages(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	underTest, client := setupTest(ctx, t)

	client.EXPECT().GetPages(gomock.Any()).Return(nil, nil)

	_, err := underTest.Read(ctx)
	is.True(errors.Is(err, sdk.ErrBackoffRetry))
}

func TestSource_Read_PagesSameTimestamp(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	underTest, cl := setupTest(ctx, t)

	// pages which were fetched in the same minute
	// in which they were last edited have special treatment
	// Also see: TestSource_Read_FreshPages_PositionNotSaved
	lastEdited := time.Now().Add(-time.Hour)
	p1 := client.Page{ID: uuid.New().String(), LastEditedTime: lastEdited}
	p2 := client.Page{ID: uuid.New().String(), LastEditedTime: lastEdited}

	cl.EXPECT().GetPages(gomock.Any()).Return([]client.Page{p1, p2}, nil)
	cl.EXPECT().GetPage(gomock.Any(), p1.ID).Return(p1, nil)

	// the position should not contain a timestamp
	// as we didn't read page p2 which is from the same minute
	wantPos, err := position{ID: p1.ID}.toSDKPosition()
	is.NoErr(err)

	r1, err := underTest.Read(ctx)
	is.NoErr(err)
	is.Equal(wantPos, r1.Position)
}

func TestSource_Read_PageNotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	underTest, cl := setupTest(ctx, t)

	p := client.Page{
		ID:             uuid.New().String(),
		LastEditedTime: time.Now(),
	}
	cl.EXPECT().GetPages(gomock.Any()).Return([]client.Page{p}, nil)
	cl.EXPECT().GetPage(gomock.Any(), p.ID).Return(p, client.ErrPageNotFound)

	_, err := underTest.Read(ctx)
	is.True(errors.Is(err, sdk.ErrBackoffRetry))
}

func TestSource_Read_FreshPages_PositionNotSaved(t *testing.T) {
	// For more information about why we have this test,
	// see discussion in docs/cdc.md
	is := is.New(t)
	ctx := context.Background()
	underTest, cl := setupTest(ctx, t)

	// todo make sure that reading the pages happens in the same minute
	p1 := client.Page{
		ID:             uuid.New().String(),
		LastEditedTime: time.Now(),
	}
	p2 := client.Page{
		ID:             uuid.New().String(),
		LastEditedTime: time.Now(),
	}

	cl.EXPECT().GetPages(gomock.Any()).Return([]client.Page{p1, p2}, nil)
	cl.EXPECT().GetPage(gomock.Any(), p1.ID).Return(p1, nil)
	cl.EXPECT().GetPage(gomock.Any(), p2.ID).Return(p2, nil)

	// Both resulting records should have NO timestamp in their position
	// We use two pages and two records, as there's some logic relying
	// on the number of page IDs.
	for i := 0; i < 2; i++ {
		rec, err := underTest.Read(ctx)
		is.NoErr(err)

		got, err := fromSDKPosition(rec.Position)
		is.NoErr(err)
		is.True(got.LastEditedTime.IsZero())
	}
}

func setupTest(ctx context.Context, t *testing.T) (*Source, *mock.Client) {
	is := is.New(t)

	token := "irrelevant-token"
	client := mock.NewClient(gomock.NewController(t))
	client.EXPECT().Init(token)
	underTest := NewSourceWithClient(client)
	err := underTest.Configure(ctx, map[string]string{Token: token})
	is.NoErr(err)
	err = underTest.Open(ctx, nil)
	is.NoErr(err)

	return underTest.(*Source), client
}
