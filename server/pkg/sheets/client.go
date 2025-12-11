package sheets

import (
	"context"
	"fmt"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Client struct {
	service *sheets.Service
}

type Config struct {
	CredentialsPath string
	CredentialsJSON []byte
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	var opts []option.ClientOption

	if cfg.CredentialsPath != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsPath))
	} else if len(cfg.CredentialsJSON) > 0 {
		opts = append(opts, option.WithCredentialsJSON(cfg.CredentialsJSON))
	} else {
		return nil, fmt.Errorf("sheets: credentials path or JSON is required")
	}

	service, err := sheets.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("sheets: failed to create service: %w", err)
	}

	return &Client{
		service: service,
	}, nil
}

func (c *Client) Service() *sheets.Service {
	return c.service
}

func (c *Client) AppendValues(ctx context.Context, spreadsheetID, range_ string, values [][]interface{}) error {
	if c.service == nil {
		return fmt.Errorf("sheets: service is nil")
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	_, err := c.service.Spreadsheets.Values.Append(spreadsheetID, range_, valueRange).
		ValueInputOption("RAW").
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do()

	return err
}

func (c *Client) UpdateValues(ctx context.Context, spreadsheetID, range_ string, values [][]interface{}) error {
	if c.service == nil {
		return fmt.Errorf("sheets: service is nil")
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	_, err := c.service.Spreadsheets.Values.Update(spreadsheetID, range_, valueRange).
		ValueInputOption("RAW").
		Context(ctx).
		Do()

	return err
}

func (c *Client) ClearValues(ctx context.Context, spreadsheetID, range_ string) error {
	if c.service == nil {
		return fmt.Errorf("sheets: service is nil")
	}

	_, err := c.service.Spreadsheets.Values.Clear(spreadsheetID, range_, &sheets.ClearValuesRequest{}).Context(ctx).Do()
	return err
}
