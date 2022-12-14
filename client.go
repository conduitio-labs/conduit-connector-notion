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
	notion "github.com/conduitio-labs/notionapi"
	"net/http"
)

var (
	errPageNotFound = errors.New("page not found")
)

type defaultClient struct {
	client *notion.Client
}

func (c *defaultClient) GetPages(ctx context.Context, cursor notion.Cursor) (*notion.SearchResponse, error) {
	req := &notion.SearchRequest{
		StartCursor: cursor,
		Sort: &notion.SortObject{
			Direction: notion.SortOrderASC,
			Timestamp: notion.TimestampLastEdited,
		},
		Filter: map[string]string{
			"property": "object",
			"value":    "page",
		},
	}
	return c.client.Search.Do(ctx, req)
}

func newDefaultClient() *defaultClient {
	return &defaultClient{}
}

func (c *defaultClient) Init(token string) {
	notion.NewClient(notion.Token(token))
}

func (c *defaultClient) GetPage(ctx context.Context, id string) (*notion.Page, error) {
	page, err := c.client.Page.Get(ctx, notion.PageID(id))
	if err != nil {
		// The search endpoint that we use to list all the pages
		// can return stale results.
		// It's also possible that a page has been deleted after
		// we got the ID but before we actually read the whole page.
		if c.notFound(err) {
			return nil, fmt.Errorf("page %v: %w", id, errPageNotFound)
		}

		return nil, fmt.Errorf("failed fetching page %v: %w", id, err)
	}

	children, err := c.getChildren(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed fetching content for %v: %w", id, err)
	}
	return page, err
}

func (c *defaultClient) notFound(err error) bool {
	nErr, ok := err.(*notion.Error)
	if !ok {
		return false
	}
	return nErr.Status == http.StatusNotFound
}

// getChildren gets all the child and grand-child blocks of the input block
func (c *defaultClient) getChildren(ctx context.Context, blockID string) ([]notion.Block, error) {
	var children []notion.Block

	fetch := true
	var cursor notion.Cursor
	for fetch {
		resp, err := c.client.Block.GetChildren(
			ctx,
			notion.BlockID(blockID),
			&notion.Pagination{
				StartCursor: cursor,
			},
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed getting children for block ID %v, cursor %v: %w",
				blockID,
				cursor,
				err,
			)
		}

		// get grandchildren as well
		for _, child := range resp.Results {
			children = append(children, child)
			grandChildren, err := c.getChildren(ctx, child.GetID().String())
			if err != nil {
				return nil, err
			}
			children = append(children, grandChildren...)
		}

		fetch = resp.HasMore
		cursor = notion.Cursor(resp.NextCursor)
	}
	return children, nil
}
