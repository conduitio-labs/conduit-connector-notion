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
	"strconv"
	"time"

	sdk "github.com/conduitio/conduit-connector-sdk"
	notion "github.com/jomei/notionapi"
)

type Source struct {
	sdk.UnimplementedSource

	config         Config
	client         *notion.Client
	lastEditedTime time.Time
	fetchIDs       []string
	lastFetch      time.Time
}

func NewSource() sdk.Source {
	return &Source{}
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

func (s *Source) initPosition(pos sdk.Position) error {
	if len(pos) == 0 {
		return nil
	}
	posParsed, err := time.Parse(string(pos), time.RFC3339)
	if err != nil {
		return fmt.Errorf("failed parsing time string %v: %w", string(pos), err)
	}
	s.lastEditedTime = posParsed
	return nil
}

func (s *Source) Read(ctx context.Context) (sdk.Record, error) {
	err := s.populateIDs(ctx)
	if err != nil {
		return sdk.Record{}, fmt.Errorf("failed fetching page IDs: %w", err)
	}
	if len(s.fetchIDs) == 0 {
		return sdk.Record{}, sdk.ErrBackoffRetry
	}

	return s.nextObject(ctx)
}

func (s *Source) nextObject(ctx context.Context) (sdk.Record, error) {
	if len(s.fetchIDs) == 0 {
		return sdk.Record{}, errors.New("no page IDs available")
	}
	id := s.fetchIDs[0]
	s.fetchIDs = s.fetchIDs[1:]

	sdk.Logger(ctx).Debug().
		Str("block_id", id).
		Msg("fetching block")

	// fetch the block and then all of its children
	block, err := s.client.Block.Get(ctx, notion.BlockID(id))
	if err != nil {
		return sdk.Record{}, fmt.Errorf("failed fetching block %v: %w", id, err)
	}

	children, err := s.getChildren(ctx, block)
	if err != nil {
		return sdk.Record{}, fmt.Errorf("failed fetching blocks for %v: %w", id, err)
	}

	// Transform the block and all of its children
	// into a Conduit record.
	record, err := s.blockToRecord(ctx, block, children)
	if err != nil {
		return sdk.Record{}, fmt.Errorf("failed transforming block %v to record: %w", id, err)
	}

	s.savePosition(*block.GetLastEditedTime())
	return record, nil
}

// getChildren gets all the child and grand-child blocks of the input block
func (s *Source) getChildren(ctx context.Context, block notion.Block) ([]notion.Block, error) {
	var children []notion.Block
	if !block.GetHasChildren() {
		sdk.Logger(ctx).Debug().
			Str("block_id", block.GetID().String()).
			Msg("block has no children")
		return children, nil
	}

	fetch := true
	var cursor notion.Cursor
	for fetch {
		resp, err := s.client.Block.GetChildren(
			ctx,
			block.GetID(),
			&notion.Pagination{
				StartCursor: cursor,
			},
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed getting children for block ID %v, cursor %v: %w",
				block.GetID(),
				cursor,
				err,
			)
		}

		// get grandchildren as well
		for _, child := range resp.Results {
			children = append(children, child)
			grandChildren, err := s.getChildren(ctx, child)
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
	if !s.lastFetch.IsZero() {
		sdk.Logger(ctx).Debug().
			Dur("poll_interval", s.config.pollInterval).
			Msg("sleeping before checking for changes")
		time.Sleep(s.config.pollInterval)
	}
	s.lastFetch = time.Now()

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

func (s *Source) blockToRecord(ctx context.Context, parent notion.Block, children notion.Blocks) (sdk.Record, error) {
	payload, err := s.getPayload(ctx, children)
	if err != nil {
		return sdk.Record{}, fmt.Errorf("failed getting payload: %w", err)
	}

	return sdk.Record{
		Position:  s.getPosition(),
		Metadata:  nil,
		CreatedAt: time.Now(),
		Key:       sdk.RawData(parent.GetID().String()),
		Payload:   payload,
	}, nil
}

func (s *Source) getPosition() sdk.Position {
	return sdk.Position(
		strconv.FormatInt(s.lastEditedTime.Unix(), 10),
	)
}

func (s *Source) getPayload(ctx context.Context, children notion.Blocks) (sdk.RawData, error) {
	var payload string
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
		payload += text + "\n"
	}

	return sdk.RawData(payload), nil
}

// savePosition saves the position, if it's safe to do so.
func (s *Source) savePosition(lastEditedTime time.Time) {
	// The precision of a page's last_edited_time field is in minutes.
	// Hence, to save it as a position (from which we can safely resume
	// reading new records), we need to be sure that all pages from
	// that minute have been read.

	// todo instead of check the queue of IDs to fetch
	// we can check the respective pages' last_edited_times
	// and make sure nothing is left from lastEditedTime's minute.
	if lastEditedTime.Before(s.lastFetch) && len(s.fetchIDs) == 0 {
		s.lastEditedTime = lastEditedTime
	}
}
