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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	notion "github.com/conduitio-labs/notionapi"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

var (
	ErrPageNotFound = errors.New("page not found")
)

type Page struct {
	ID string
	// Parent is a JSON string describing the page's parent object
	Parent string
	URL    string
	// CreatedBy is a JSON string describing the user who created this page
	CreatedBy   string
	CreatedTime time.Time
	// LastEditedBy is a JSON string describing the user who last edited this page
	LastEditedBy   string
	LastEditedTime time.Time
	Archived       bool

	properties notion.Properties
	children   []notion.Block
}

func NewPage(pg *notion.Page, children []notion.Block) Page {
	return Page{
		ID:             pg.ID.String(),
		Parent:         toJSON(pg.Parent),
		URL:            pg.URL,
		CreatedTime:    pg.CreatedTime,
		CreatedBy:      toJSON(pg.CreatedBy),
		LastEditedBy:   toJSON(pg.LastEditedBy),
		LastEditedTime: pg.LastEditedTime,
		Archived:       pg.Archived,
		properties:     pg.Properties,
		children:       children,
	}
}

// PlainText returns a plain text representation of a page
func (p Page) PlainText(ctx context.Context) (string, error) {
	var plainText string
	for _, c := range p.children {
		text, err := extractText(c)
		if errors.Is(err, errNoExtractor) {
			sdk.Logger(ctx).Warn().
				Str("block_type", c.GetType().String()).
				Msg("no text extractor registered")
			continue
		}
		if err != nil {
			return "", err
		}
		plainText += text + "\n"
	}

	return plainText, nil
}

// Title returns a page's title.
func (p Page) Title() string {
	if len(p.properties) == 0 {
		return ""
	}

	tp, ok := p.properties["title"].(*notion.TitleProperty)
	if !ok || len(tp.Title) == 0 {
		return ""
	}

	return tp.Title[0].PlainText
}

type DefaultClient struct {
	client *notion.Client
}

func New() *DefaultClient {
	return &DefaultClient{}
}

func (c *DefaultClient) Init(token string) {
	c.client = notion.NewClient(notion.Token(token))
}

func (c *DefaultClient) GetPage(ctx context.Context, id string) (Page, error) {
	pg, err := c.client.Page.Get(ctx, notion.PageID(id))
	if err != nil {
		// The search endpoint that we use to list all the pages
		// can return stale results.
		// It's also possible that a page has been deleted after
		// we got the ID but before we actually read the whole page.
		if c.notFound(err) {
			return Page{}, fmt.Errorf("page %v: %w", id, ErrPageNotFound)
		}

		return Page{}, fmt.Errorf("failed fetching page %v: %w", id, err)
	}

	// fetch the page block and then all of its children
	block, err := c.client.Block.Get(ctx, notion.BlockID(id))
	if err != nil {
		if c.notFound(err) {
			return Page{}, fmt.Errorf("page %v: %w", id, ErrPageNotFound)
		}

		return Page{}, fmt.Errorf("failed fetching page block %v: %w", id, err)
	}
	if block.GetType() == notion.BlockTypeUnsupported {
		// skip children of unsupported block types
		sdk.Logger(ctx).Warn().
			Str("block_type", block.GetType().String()).
			Str("block_id", block.GetID().String()).
			Msg("skipping children of unsupported block")
		return NewPage(pg, nil), nil
	}

	children, err := c.getChildren(ctx, id)
	if err != nil {
		return Page{}, fmt.Errorf("failed fetching content for %v: %w", id, err)
	}
	return NewPage(pg, children), nil
}

func (c *DefaultClient) notFound(err error) bool {
	nErr, ok := err.(*notion.Error)
	if !ok {
		return false
	}
	return nErr.Status == http.StatusNotFound
}

// getChildren gets all the child and grand-child blocks of the input block
func (c *DefaultClient) getChildren(ctx context.Context, blockID string) ([]notion.Block, error) {
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

func (c *DefaultClient) GetPages(ctx context.Context) ([]Page, error) {
	var allPages []Page

	fetch := true
	var cursor notion.Cursor
	for fetch {
		response, err := c.searchPages(ctx, cursor)
		if err != nil {
			return nil, fmt.Errorf("search failed: %w", err)
		}

		sdk.Logger(ctx).Debug().Msgf("got search response with %v results", len(response.Results))
		pages, err := c.toPages(response.Results)
		if err != nil {
			return nil, fmt.Errorf("failed to transformed the pages: %w", err)
		}

		sdk.Logger(ctx).Info().Msgf("c.toPages returned %v pages", len(pages))
		allPages = append(allPages, pages...)

		fetch = response.HasMore
		cursor = response.NextCursor
	}

	sdk.Logger(ctx).Info().Msgf("GetPages: %v pages", len(allPages))
	return allPages, nil
}

func (c *DefaultClient) searchPages(ctx context.Context, cursor notion.Cursor) (*notion.SearchResponse, error) {
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
	response, err := c.client.Search.Do(ctx, req)
	return response, err
}

func (c *DefaultClient) toPages(results []notion.Object) ([]Page, error) {
	pages := make([]Page, len(results))
	for i, res := range results {
		if strings.ToLower(res.GetObject().String()) != "page" {
			// shouldn't ever happen, as we requested only the pages in the search method.
			return nil, fmt.Errorf("got unexpected object %q in search results", res.GetObject().String())
		}
		pages[i] = NewPage(res.(*notion.Page), nil)
	}

	return pages, nil
}

// toJSON converts `v` into a JSON string.
// In case that's not possible, the function returns an empty string.
func toJSON(v any) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(bytes)
}
