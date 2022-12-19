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

package client

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/conduitio-labs/conduit-connector-notion/client/mock"
	notion "github.com/conduitio-labs/notionapi"
	"github.com/golang/mock/gomock"
	"github.com/matryer/is"
)

func TestClient_GetPage(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	id := "test-page-id"
	ctrl := gomock.NewController(t)
	page := mock.NewPageService(ctrl)
	block := mock.NewBlockService(ctrl)

	underTest := New()
	underTest.client = &notion.Client{
		Page:  page,
		Block: block,
	}

	notionPage := &notion.Page{ID: notion.ObjectID(id)}
	page.EXPECT().Get(gomock.Any(), notion.PageID(id)).
		Return(notionPage, nil)
	block.EXPECT().GetChildren(gomock.Any(), notion.BlockID(id), gomock.Any()).
		Return(&notion.GetChildrenResponse{}, nil)

	want := NewPage(notionPage, nil)
	got, err := underTest.GetPage(ctx, id)
	is.NoErr(err)
	is.Equal(want, got)
}

func TestClient_GetPage_NotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	id := "test-page-id"
	ctrl := gomock.NewController(t)
	page := mock.NewPageService(ctrl)

	underTest := New()
	underTest.client = &notion.Client{
		Page: page,
	}

	page.EXPECT().Get(gomock.Any(), notion.PageID(id)).
		Return(nil, &notion.Error{Status: http.StatusNotFound})

	_, err := underTest.GetPage(ctx, id)
	is.True(errors.Is(err, ErrPageNotFound))
}

func TestClient_GetPages_Empty(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	search := mock.NewSearchService(ctrl)

	underTest := New()
	underTest.client = &notion.Client{
		Search: search,
	}

	req := pageSearchRequest()
	search.EXPECT().Do(gomock.Any(), req).
		Return(&notion.SearchResponse{}, nil)

	pages, err := underTest.GetPages(ctx, time.Time{})
	is.NoErr(err)
	is.True(len(pages) == 0)
}

func TestClient_GetPages_FilterByTimestamp(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	search := mock.NewSearchService(ctrl)

	underTest := New()
	underTest.client = &notion.Client{
		Search: search,
	}

	req := pageSearchRequest()
	resp := &notion.SearchResponse{
		Results: []notion.Object{
			&notion.Page{
				ID:             "page-1",
				LastEditedTime: time.Now().Add(-2 * time.Hour),
			},
			&notion.Page{
				ID:             "page-2",
				LastEditedTime: time.Now(),
			},
		},
	}
	search.EXPECT().Do(gomock.Any(), req).
		Return(resp, nil)

	pages, err := underTest.GetPages(ctx, time.Now().Add(-time.Hour))
	is.NoErr(err)
	is.Equal(1, len(pages))
	is.Equal("page-2", pages[0].ID)
}

func pageSearchRequest() *notion.SearchRequest {
	return &notion.SearchRequest{
		StartCursor: "",
		Sort: &notion.SortObject{
			Direction: notion.SortOrderASC,
			Timestamp: notion.TimestampLastEdited,
		},
		Filter: map[string]string{
			"property": "object",
			"value":    "page",
		},
	}
}
