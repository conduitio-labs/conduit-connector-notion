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

type notionBlock struct {
	blockType string         `json:"type"`
	children  []notion.Block `json:"children"`
}

type Source struct {
	sdk.UnimplementedSource

	config         Config
	client         *notion.Client
	lastEditedTime time.Time
	fetchIDs       []string
}

func NewSource() sdk.Source {
	return &Source{}
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
		// todo configure poll period
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
	// check if HasChildren
	children, err := s.client.Block.GetChildren(ctx, notion.BlockID(pageID), nil)
	if err != nil {
		return sdk.Record{}, fmt.Errorf("failed fetching blocks for %v: %w", pageID, err)
	}

	record, err := s.blockToRecord(ctx, block, children.Results)
	if err != nil {
		return sdk.Record{}, err
	}
	s.lastEditedTime = *block.GetLastEditedTime()
	return record, nil
}

func (s *Source) Ack(ctx context.Context, position sdk.Position) error {
	return nil
}

func (s *Source) Teardown(ctx context.Context) error {
	return nil
}

func (s *Source) populateIDs(ctx context.Context) error {
	if len(s.fetchIDs) > 0 {
		return nil
	}

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
	}
	return s.client.Search.Do(ctx, req)
}

func (s *Source) blockToRecord(ctx context.Context, b notion.Block, cb notion.Blocks) (sdk.Record, error) {
	nb := notionBlock{
		blockType: b.GetType().String(),
		children:  cb,
	}
	// todo remove indent
	payload, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return sdk.Record{}, fmt.Errorf("failed marshalling payload: %w", err)
	}

	sdk.Logger(ctx).Debug().Msgf("payload %v", nb)
	return sdk.Record{
		Position:  s.getKey(b),
		Metadata:  nil,
		CreatedAt: time.Now(),
		Key:       sdk.RawData(b.GetID().String()),
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
