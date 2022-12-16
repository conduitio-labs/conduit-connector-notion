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

//go:generate mockgen -destination=mock/client.go -package=mock -mock_names=Client=Client . Client

package notion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/conduitio-labs/conduit-connector-notion/client"
	sdk "github.com/conduitio/conduit-connector-sdk"
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

func fromSDKPosition(sdkPos sdk.Position) (position, error) {
	pos := position{}
	err := json.Unmarshal(sdkPos, &pos)
	if err != nil {
		return position{}, fmt.Errorf("failed unmarshalling position: %w", err)
	}
	return pos, nil
}

type recordPayload struct {
	Plaintext string            `json:"plaintext"`
	Metadata  map[string]string `json:"metadata"`
}

type Client interface {
	// GetPage gets a page with given ID
	GetPage(ctx context.Context, id string) (client.Page, error)
	// Init initializes the client with the given access token
	Init(token string)
	// GetPages returns *all* pages in Notion
	GetPages(ctx context.Context) ([]client.Page, error)
}

type Source struct {
	sdk.UnimplementedSource

	config Config
	client Client
	// lastMinuteRead is the last minute from which we
	// processed all pages
	lastMinuteRead time.Time
	// fetchIDs contains IDs of pages which need to be fetched
	fetchIDs []string
	// lastPoll is the time at which we polled Notion the last time
	lastPoll time.Time
}

func NewSource() sdk.Source {
	return NewSourceWithClient(client.New())
}

func NewSourceWithClient(c Client) sdk.Source {
	return &Source{client: c}
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
	s.client.Init(s.config.token)
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

	pos, err := fromSDKPosition(sdkPos)
	if err != nil {
		return err
	}
	s.lastMinuteRead = pos.LastEditedTime

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
	pg, err := s.client.GetPage(ctx, id)
	// The search endpoint that we use to list all the pages
	// can return stale results.
	// It's also possible that a page has been deleted after
	// we got the ID but before we actually read the whole page.
	if errors.Is(err, client.ErrPageNotFound) {
		sdk.Logger(ctx).Info().
			Str("block_id", id).
			Msg("the resource does not exist or the resource has not been shared with owner of the token")

		return s.nextPage(ctx)
	}
	if err != nil {
		return sdk.Record{}, fmt.Errorf("failed fetching page %v: %w", id, err)
	}

	record, err := s.pageToRecord(ctx, pg)
	if err != nil {
		return sdk.Record{}, fmt.Errorf("failed transforming page %v to record: %w", id, err)
	}

	s.savePosition(pg.LastEditedTime)
	pos, err := s.getPosition(pg)
	if err != nil {
		return sdk.Record{}, err
	}
	record.Position = pos
	return record, nil
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

	// We don't want to sleep before the first poll attempt
	if !s.lastPoll.IsZero() {
		sdk.Logger(ctx).Debug().
			Dur("poll_interval", s.config.pollInterval).
			Msg("sleeping before checking for changes")
		time.Sleep(s.config.pollInterval)
	}
	// todo maybe get the time at which the search was performed from the client
	// that will make it possible to test it better
	pollTime := time.Now()

	sdk.Logger(ctx).Debug().Msg("populating IDs")
	allPages, err := s.client.GetPages(ctx)
	if err != nil {
		return fmt.Errorf("failed getting changed pages: %w", err)
	}
	// we can set s.lastPoll only when a search succeeds
	// otherwise, we might miss changes in the next succeeding search
	s.lastPoll = pollTime

	s.addToFetchIDs(ctx, allPages)
	sdk.Logger(ctx).Debug().Msgf("fetched %v IDs", len(s.fetchIDs))

	return nil
}

func (s *Source) addToFetchIDs(ctx context.Context, pages []client.Page) {
	sdk.Logger(ctx).Debug().
		Msgf("checking %v pages for changes", len(pages))

	for _, pg := range pages {
		sdk.Logger(ctx).Trace().
			Str("page_id", pg.ID).
			Time("last_edited_time", pg.LastEditedTime).
			Time("created_time", pg.CreatedTime).
			Msg("checking if page has changed")

		// todo move the check to the client
		if pg.LastEditedTime.After(s.lastMinuteRead) {
			s.fetchIDs = append(s.fetchIDs, pg.ID)
		}
	}
}

func (s *Source) pageToRecord(ctx context.Context, pg client.Page) (sdk.Record, error) {
	payload, err := s.getPayload(ctx, pg)
	if err != nil {
		return sdk.Record{}, fmt.Errorf("failed getting payload: %w", err)
	}

	return sdk.Record{
		Metadata:  nil,
		CreatedAt: time.Now(),
		Key:       sdk.RawData(pg.ID),
		Payload:   payload,
	}, nil
}

func (s *Source) getPosition(pg client.Page) (sdk.Position, error) {
	return position{
		ID:             pg.ID,
		LastEditedTime: s.lastMinuteRead,
	}.toSDKPosition()
}

func (s *Source) getPayload(ctx context.Context, pg client.Page) (sdk.RawData, error) {
	plainText, err := pg.PlainText(ctx)
	if err != nil {
		return nil, err
	}
	payload := recordPayload{
		Plaintext: plainText,
		Metadata:  s.getMetadata(pg),
	}
	return json.Marshal(payload)
}

func (s *Source) getMetadata(pg client.Page) map[string]string {
	return map[string]string{
		"notion.title":          pg.Title(),
		"notion.url":            pg.URL,
		"notion.createdTime":    pg.CreatedTime.Format(time.RFC3339),
		"notion.lastEditedTime": pg.LastEditedTime.Format(time.RFC3339),
		"notion.createdBy":      pg.CreatedBy,
		"notion.lastEditedBy":   pg.LastEditedBy,
		"notion.archived":       strconv.FormatBool(pg.Archived),
		"notion.parent":         pg.Parent,
	}
}

// savePosition saves the position, if it's safe to do so.
func (s *Source) savePosition(t time.Time) {
	// see discussion in docs/cdc.md
	lastTopMinute := time.Now().Truncate(time.Minute)
	if t.After(lastTopMinute) {
		return
	}
	// The precision of a page's last_edited_time field is in minutes.
	// Hence, to save it as a position (from which we can safely resume
	// reading new records), we need to be sure that all pages from
	// that minute have been read.

	// todo instead of check the queue of IDs to fetch
	// we can check the respective pages' last_edited_times
	// and make sure nothing is left from `lastMinuteRead`.
	if t.Before(s.lastPoll) && len(s.fetchIDs) == 0 {
		s.lastMinuteRead = t
	}
}
