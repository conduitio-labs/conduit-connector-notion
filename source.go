package notion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sdk "github.com/conduitio/conduit-connector-sdk"
	notion "github.com/jomei/notionapi"
)

type NotionBlock struct {
	BlockType string         `json:"type"`
	Children  []notion.Block `json:"children"`
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
			Default:     "localhost:10000",
			Required:    true,
			Description: "The URL of the server.",
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
	pageID := s.fetchIDs[0]
	s.fetchIDs = s.fetchIDs[1:]

	// todo support databases
	sdk.Logger(ctx).Debug().
		Str("page_id", pageID).
		Msg("fetching page")

	block, err := s.client.Block.Get(ctx, notion.BlockID(pageID))
	if err != nil {
		return sdk.Record{}, fmt.Errorf("failed fetching page %v: %w", pageID, err)
	}
	// todo support grand-children
	children, err := s.getChildren(ctx, block)
	if err != nil {
		return sdk.Record{}, fmt.Errorf("failed fetching blocks for %v: %w", pageID, err)
	}

	record, err := s.blockToRecord(block, children)
	if err != nil {
		return sdk.Record{}, err
	}
	s.lastEditedTime = *block.GetLastEditedTime()
	return record, nil
}

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

		children = append(children, resp.Results...)

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
			if page.LastEditedTime.After(s.lastEditedTime) {
				s.fetchIDs = append(s.fetchIDs, page.ID.String())
			}
		default:
			sdk.Logger(ctx).Warn().
				Str("object_type", result.GetObject().String()).
				Msg("object type currently not supported")
		}
	}
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

func (s *Source) blockToRecord(parent notion.Block, children notion.Blocks) (sdk.Record, error) {
	nb := NotionBlock{
		BlockType: parent.GetType().String(),
		Children:  children,
	}
	payload, err := json.Marshal(nb)
	if err != nil {
		return sdk.Record{}, fmt.Errorf("failed marshalling payload: %w", err)
	}

	return sdk.Record{
		Position:  s.getKey(parent),
		Metadata:  nil,
		CreatedAt: time.Now(),
		Key:       sdk.RawData(parent.GetID().String()),
		Payload:   sdk.RawData(payload),
	}, nil
}

func (s *Source) getKey(b notion.Block) sdk.Position {
	return sdk.Position(b.GetLastEditedTime().Format(time.RFC3339))
}

func (s *Source) getTitle(page *notion.Page) string {
	title := page.Properties["title"]
	if title != nil {
		texts := title.(*notion.TitleProperty).Title
		for _, text := range texts {
			return text.PlainText
		}
	}
	return ""
}
