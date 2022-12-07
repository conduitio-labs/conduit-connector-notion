package notion

import (
	"context"
	"fmt"
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
	posParsed, err := time.Parse(string(pos), time.RFC3339)
	if err != nil {
		return fmt.Errorf("invalid position %v: %w", string(pos), err)
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

	return sdk.Record{}, nil
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

	return nil
}

func (s *Source) addToFetchIDs(ctx context.Context, results *notion.SearchResponse) {
	for _, result := range results.Results {
		switch result.GetObject().String() {
		case "page":
			s.fetchIDs = append(s.fetchIDs, result.(*notion.Page).ID.String())
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
