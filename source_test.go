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
	"testing"
	"time"

	"github.com/conduitio-labs/conduit-connector-notion/client"
	"github.com/conduitio-labs/conduit-connector-notion/mock"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
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
	ctx := context.Background()
	underTest, cl := setupTest(ctx, t, nil)
	cl.EXPECT().GetPages(ctx, zeroTimeMatcher{}).
		Return(nil, nil)

	_, err := underTest.Read(ctx)
	is.True(errors.Is(err, sdk.ErrBackoffRetry))
}

func TestSource_Open_WithPosition(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	pos := position{
		ID:             "test-id",
		LastEditedTime: time.Now(),
	}
	sdkPos, err := pos.toSDKPosition()
	is.NoErr(err)

	underTest, cl := setupTest(ctx, t, sdkPos)
	cl.EXPECT().GetPages(ctx, timeEqMatcher{pos.LastEditedTime}).
		Return(nil, nil)

	_, err = underTest.Read(ctx)
	is.True(errors.Is(err, sdk.ErrBackoffRetry))
}

func TestSource_Read_NoPages(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	underTest, client := setupTest(ctx, t, nil)

	client.EXPECT().
		GetPages(gomock.Any(), zeroTimeMatcher{}).
		Return(nil, nil)

	_, err := underTest.Read(ctx)
	is.True(errors.Is(err, sdk.ErrBackoffRetry))
}

func TestSource_Read_SinglePage(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	underTest, cl := setupTest(ctx, t, nil)

	// pages which were fetched in the same minute
	// in which they were last edited have special treatment
	// Also see: TestSource_Read_FreshPages_PositionNotSaved
	lastEdited := time.Now().Add(-time.Hour)
	p := client.Page{ID: uuid.New().String(), LastEditedTime: lastEdited}

	cl.EXPECT().GetPages(gomock.Any(), zeroTimeMatcher{}).
		Return([]client.Page{p}, nil)
	cl.EXPECT().GetPage(gomock.Any(), p.ID).
		Return(p, nil)

	// the position should contain a timestamp
	rec, err := underTest.Read(ctx)
	is.NoErr(err)

	gotPos, err := fromSDKPosition(rec.Position)
	is.NoErr(err)
	is.True(p.LastEditedTime.Equal(gotPos.LastEditedTime))
}

func TestSource_Read_PagesSameTimestamp(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	underTest, cl := setupTest(ctx, t, nil)

	// pages which were fetched in the same minute
	// in which they were last edited have special treatment
	// Also see: TestSource_Read_FreshPages_PositionNotSaved
	lastEdited := time.Now().Add(-time.Hour)
	p1 := client.Page{ID: uuid.New().String(), LastEditedTime: lastEdited}
	p2 := client.Page{ID: uuid.New().String(), LastEditedTime: lastEdited}

	cl.EXPECT().GetPages(gomock.Any(), zeroTimeMatcher{}).
		Return([]client.Page{p1, p2}, nil)
	cl.EXPECT().GetPage(gomock.Any(), p1.ID).
		Return(p1, nil)

	// the position should NOT contain a timestamp
	// as we didn't read page p2 which is from the same minute
	rec, err := underTest.Read(ctx)
	is.NoErr(err)

	gotPos, err := fromSDKPosition(rec.Position)
	is.NoErr(err)
	is.True(gotPos.LastEditedTime.IsZero())
}

func TestSource_Read_PagesDifferentTimestamps(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	underTest, cl := setupTest(ctx, t, nil)

	p1 := client.Page{ID: uuid.New().String(), LastEditedTime: time.Now().Add(-2 * time.Hour)}
	p2 := client.Page{ID: uuid.New().String(), LastEditedTime: time.Now().Add(-time.Hour)}

	cl.EXPECT().GetPages(gomock.Any(), zeroTimeMatcher{}).
		Return([]client.Page{p1, p2}, nil)
	cl.EXPECT().GetPage(gomock.Any(), p1.ID).
		Return(p1, nil)
	cl.EXPECT().GetPage(gomock.Any(), p2.ID).
		Return(p2, nil)

	// the position should contain a timestamp
	// as we page p2 which is NOT from the same minute as p1
	rec1, err := underTest.Read(ctx)
	is.NoErr(err)

	pos1, err := fromSDKPosition(rec1.Position)
	is.NoErr(err)
	is.True(pos1.LastEditedTime.Equal(p1.LastEditedTime))

	rec2, err := underTest.Read(ctx)
	is.NoErr(err)

	pos2, err := fromSDKPosition(rec2.Position)
	is.NoErr(err)
	is.True(pos2.LastEditedTime.Equal(p2.LastEditedTime))
}

func TestSource_Read_FreshPages_PositionNotSaved(t *testing.T) {
	// For more information about why we have this test,
	// see discussion in docs/cdc.md
	is := is.New(t)
	ctx := context.Background()
	underTest, cl := setupTest(ctx, t, nil)

	// todo make sure that reading the pages happens
	//   in the same minute in which they are last edited
	// set up test pages
	count := 2
	pages := make([]client.Page, count)
	for i := 0; i < count; i++ {
		p := client.Page{ID: uuid.New().String(), LastEditedTime: time.Now()}
		pages[i] = p
		cl.EXPECT().GetPage(gomock.Any(), p.ID).
			Return(p, nil)
	}
	cl.EXPECT().GetPages(gomock.Any(), zeroTimeMatcher{}).
		Return(pages, nil)

	// Both resulting records should have NO timestamp in their position
	// We use two pages and two records, as there's some logic relying
	// on the number of page IDs.
	for i := 0; i < count; i++ {
		rec, err := underTest.Read(ctx)
		is.NoErr(err)

		got, err := fromSDKPosition(rec.Position)
		is.NoErr(err)
		is.True(got.LastEditedTime.IsZero())
	}
}

func TestSource_Read_PageNotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	underTest, cl := setupTest(ctx, t, nil)

	p := client.Page{
		ID:             uuid.New().String(),
		LastEditedTime: time.Now(),
	}
	cl.EXPECT().GetPages(gomock.Any(), zeroTimeMatcher{}).
		Return([]client.Page{p}, nil)
	cl.EXPECT().GetPage(gomock.Any(), p.ID).
		Return(p, client.ErrPageNotFound)

	_, err := underTest.Read(ctx)
	is.True(errors.Is(err, sdk.ErrBackoffRetry))
}

func TestSource_Read_GetPageError(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	underTest, cl := setupTest(ctx, t, nil)

	p := client.Page{
		ID:             uuid.New().String(),
		LastEditedTime: time.Now(),
	}
	pageErr := errors.New("lazy service error")
	cl.EXPECT().GetPages(gomock.Any(), zeroTimeMatcher{}).
		Return([]client.Page{p}, nil)
	cl.EXPECT().GetPage(gomock.Any(), p.ID).
		Return(client.Page{}, pageErr)

	_, err := underTest.Read(ctx)
	is.True(errors.Is(err, pageErr))
}

func TestSource_Read_SearchFailed(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	underTest, cl := setupTest(ctx, t, nil)

	searchErr := errors.New("search failed successfully")
	cl.EXPECT().GetPages(gomock.Any(), zeroTimeMatcher{}).
		Return(nil, searchErr)

	_, err := underTest.Read(ctx)
	is.True(errors.Is(err, searchErr))
}

func setupTest(ctx context.Context, t *testing.T, pos sdk.Position) (*Source, *mock.Client) {
	is := is.New(t)

	token := "irrelevant-token"
	client := mock.NewClient(gomock.NewController(t))
	client.EXPECT().Init(token)
	underTest := NewSourceWithClient(client)
	err := underTest.Configure(ctx, map[string]string{Token: token})
	is.NoErr(err)
	err = underTest.Open(ctx, pos)
	is.NoErr(err)

	return underTest.(*Source), client
}
