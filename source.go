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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	sdk "github.com/conduitio/conduit-connector-sdk"
	notion "github.com/jomei/notionapi"
)

type position struct {
	ID             string
	LastEditedTime time.Time
}

func (p position) toSDKPosition() (sdk.Position, error) {
	bytes, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed marshalling position: %w", err)
	}
	return bytes, nil
}

type recordPayload struct {
	Plaintext string            `json:"plaintext"`
	Metadata  map[string]string `json:"metadata"`
}

type Source struct {
	sdk.UnimplementedSource

	config         Config
	client         *notion.Client
	lastEditedTime time.Time
	fetchIDs       []string
	firstFetch     bool
}

func NewSource() sdk.Source {
	return &Source{firstFetch: true}
}

func (s *Source) Parameters() map[string]sdk.Parameter {
	return map[string]sdk.Parameter{
		Token: {
			Default:     "",
			Required:    true,
			Description: "Internal integration token.",
		},
		PollInterval: {
			Default:     "1 minute",
			Required:    false,
			Description: "Interval at which we poll Notion for changes. A Go duration string.",
		},
	}
}

func (s *Source) Configure(ctx context.Context, cfg map[string]string) error {
	sdk.Logger(ctx).Info().Msg("Configuring a Source Connector...")
	config, err := ParseConfig(cfg)
	if err != nil {
		return err
	}

	s.config = config
	return nil
}

func (s *Source) Open(_ context.Context, pos sdk.Position) error {
	s.client = notion.NewClient(notion.Token(s.config.token))
	err := s.initPosition(pos)
	if err != nil {
		return fmt.Errorf("failed initializing position: %w", err)
	}
	return err
}

func (s *Source) initPosition(sdkPos sdk.Position) error {
	if len(sdkPos) == 0 {
		return nil
	}

	pos, err := s.fromSDKPosition(sdkPos)
	if err != nil {
		return err
	}
	s.lastEditedTime = pos.LastEditedTime

	return nil
}

func (s *Source) Read(ctx context.Context) (sdk.Record, error) {
	err := s.populateIDs(ctx)
	if err != nil {
		return sdk.Record{}, fmt.Errorf("failed fetching page IDs: %w", err)
	}

	return s.nextPage(ctx)
}

func (s *Source) nextPage(ctx context.Context) (sdk.Record, error) {
	if len(s.fetchIDs) == 0 {
		return sdk.Record{}, sdk.ErrBackoffRetry
	}

	id := s.fetchIDs[0]
	s.fetchIDs = s.fetchIDs[1:]

	sdk.Logger(ctx).Debug().
		Str("page_id", id).
		Msg("fetching page")

	// fetch the page and then all of its children
	page, err := s.client.Page.Get(ctx, notion.PageID(id))
	if err != nil {
		// The search endpoint that we use to list all the pages
		// can return stale results.
		// It's also possible that a page has been deleted after
		// we got the ID but before we actually read the whole page.
		if s.notFound(err) {
			sdk.Logger(ctx).Info().
				Str("block_id", id).
				Msg("the resource does not exist or the resource has not been shared with owner of the token")

			return s.nextPage(ctx)
		}

		return sdk.Record{}, fmt.Errorf("failed fetching page %v: %w", id, err)
	}

	children, err := s.getChildren(ctx, id)
	if err != nil {
		return sdk.Record{}, fmt.Errorf("failed fetching content for %v: %w", id, err)
	}

	record, err := s.pageToRecord(ctx, page, children)
	if err != nil {
		return sdk.Record{}, fmt.Errorf("failed transforming page %v to record: %w", id, err)
	}

	s.lastEditedTime = page.LastEditedTime

	return record, nil
}

// getChildren gets all the child and grand-child blocks of the input block
func (s *Source) getChildren(ctx context.Context, blockID string) ([]notion.Block, error) {
	var children []notion.Block

	fetch := true
	var cursor notion.Cursor
	for fetch {
		resp, err := s.client.Block.GetChildren(
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
			grandChildren, err := s.getChildren(ctx, child.GetID().String())
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

func (s *Source) Ack(context.Context, sdk.Position) error {
	return nil
}

func (s *Source) Teardown(context.Context) error {
	return nil
}

func (s *Source) populateIDs(ctx context.Context) error {
	if len(s.fetchIDs) > 0 {
		return nil
	}
	// the first read attempt (when the connector starts)
	if !s.firstFetch {
		sdk.Logger(ctx).Debug().
			Dur("poll_interval", s.config.pollInterval).
			Msg("sleeping before checking for changes")
		time.Sleep(s.config.pollInterval)
	}
	s.firstFetch = false

	sdk.Logger(ctx).Debug().Msg("populating IDs")
	fetch := true
	var cursor notion.Cursor
	for fetch {
		results, err := s.getPages(ctx, cursor)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}
		s.addToFetchIDs(ctx, results)

		fetch = results.HasMore
		cursor = results.NextCursor
	}

	sdk.Logger(ctx).Info().Msgf("fetched %v IDs", len(s.fetchIDs))
	return nil
}

func (s *Source) addToFetchIDs(ctx context.Context, results *notion.SearchResponse) {
	for _, result := range results.Results {
		switch result.GetObject().String() {
		case "page":
			page := result.(*notion.Page)
			sdk.Logger(ctx).Trace().
				Str("page_id", page.ID.String()).
				Time("last_edited_time", page.LastEditedTime).
				Time("created_time", page.CreatedTime).
				Msg("checking if page has changed")
			if s.hasChanged(page) {
				s.fetchIDs = append(s.fetchIDs, page.ID.String())
			}
		default:
			sdk.Logger(ctx).Warn().
				Str("object_type", result.GetObject().String()).
				Msg("object type currently not supported")
		}
	}
}

func (s *Source) hasChanged(page *notion.Page) bool {
	// see discussion in docs/cdc.md
	lastTopMinute := time.Now().Truncate(time.Minute)
	return page.LastEditedTime.After(s.lastEditedTime) &&
		page.LastEditedTime.Before(lastTopMinute)
}

func (s *Source) getPages(ctx context.Context, cursor notion.Cursor) (*notion.SearchResponse, error) {
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
	return s.client.Search.Do(ctx, req)
}

func (s *Source) pageToRecord(ctx context.Context, page *notion.Page, children notion.Blocks) (sdk.Record, error) {
	payload, err := s.getPayload(ctx, children, s.getMetadata(page))
	if err != nil {
		return sdk.Record{}, fmt.Errorf("failed getting payload: %w", err)
	}

	pos, err := s.getPosition(page)
	if err != nil {
		return sdk.Record{}, err
	}
	return sdk.Record{
		Position:  pos,
		Metadata:  nil,
		CreatedAt: time.Now(),
		Key:       sdk.RawData(page.ID),
		Payload:   payload,
	}, nil
}

func (s *Source) getPosition(page *notion.Page) (sdk.Position, error) {
	if page == nil {
		return nil, nil
	}
	return position{
		ID:             page.ID.String(),
		LastEditedTime: page.LastEditedTime,
	}.toSDKPosition()
}

func (s *Source) fromSDKPosition(sdkPos sdk.Position) (position, error) {
	pos := position{}
	err := json.Unmarshal(sdkPos, &pos)
	if err != nil {
		return position{}, fmt.Errorf("failed unmarshalling position: %w", err)
	}
	return pos, nil
}

func (s *Source) getPayload(
	ctx context.Context,
	children notion.Blocks,
	metadata map[string]string,
) (sdk.RawData, error) {
	var plainText string
	for _, c := range children {
		text, err := extractText(c)
		if errors.Is(err, errNoExtractor) {
			sdk.Logger(ctx).Warn().
				Str("block_type", c.GetType().String()).
				Msg("no text extractor registered")
			continue
		}
		if err != nil {
			return nil, err
		}
		plainText += text + "\n"
	}

	payload := recordPayload{
		Plaintext: plainText,
		Metadata:  metadata,
	}
	return json.Marshal(payload)
}

func (s *Source) getMetadata(page *notion.Page) map[string]string {
	return map[string]string{
		"notion.title":          s.getPageTitle(page),
		"notion.url":            page.URL,
		"notion.createdTime":    page.CreatedTime.Format(time.RFC3339),
		"notion.lastEditedTime": page.LastEditedTime.Format(time.RFC3339),
		"notion.createdBy":      s.toJSON(page.CreatedBy),
		"notion.lastEditedBy":   s.toJSON(page.LastEditedBy),
		"notion.archived":       strconv.FormatBool(page.Archived),
		"notion.parent":         s.toJSON(page.Parent),
	}
}

// toJSON converts `v` into a JSON string.
// In case that's not possible, the function returns an empty string.
func (s *Source) toJSON(v any) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(bytes)
}

// getPageTitle returns the input page's title.
// In case that's not possible, the function returns an empty string.
func (s *Source) getPageTitle(page *notion.Page) string {
	if page == nil || len(page.Properties) == 0 {
		return ""
	}

	tp, ok := page.Properties["title"].(*notion.TitleProperty)
	if !ok || len(tp.Title) == 0 {
		return ""
	}

	return tp.Title[0].PlainText
}

func (s *Source) notFound(err error) bool {
	nErr, ok := err.(*notion.Error)
	if !ok {
		return false
	}
	return nErr.Status == http.StatusNotFound
}
