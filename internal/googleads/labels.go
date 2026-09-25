package googleads

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pbclients "github.com/shenzhencenter/google-ads-pb/clients"
	"github.com/shenzhencenter/google-ads-pb/common"
	"github.com/shenzhencenter/google-ads-pb/resources"
	"github.com/shenzhencenter/google-ads-pb/services"
)

// LabelInput is the canonical label — a named, color-tagged annotation
// reusable across campaigns, ad groups, ads, and keyword criteria via
// the four *_label join resources.
type LabelInput struct {
	CustomerID      string
	Name            string
	BackgroundColor string // HEX, e.g. "#FFAABB"
	Description     string // ≤ 200 chars
}

type LabelView struct {
	ResourceName    string
	Name            string
	Status          string
	BackgroundColor string
	Description     string
}

func (c *Client) CreateLabel(ctx context.Context, in LabelInput) (string, error) {
	cl, err := c.labelClient(ctx)
	if err != nil {
		return "", err
	}
	label := &resources.Label{Name: StringPtr(in.Name)}
	if in.BackgroundColor != "" || in.Description != "" {
		label.TextLabel = &common.TextLabel{}
		if in.BackgroundColor != "" {
			label.TextLabel.BackgroundColor = StringPtr(in.BackgroundColor)
		}
		if in.Description != "" {
			label.TextLabel.Description = StringPtr(in.Description)
		}
	}
	resp, err := cl.MutateLabels(ctx, &services.MutateLabelsRequest{
		CustomerId: in.CustomerID,
		Operations: []*services.LabelOperation{{
			Operation: &services.LabelOperation_Create{Create: label},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

func (c *Client) UpdateLabel(ctx context.Context, resourceName string, in LabelInput, paths []string) error {
	cl, err := c.labelClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "labels")
	if err != nil {
		return err
	}
	label := &resources.Label{ResourceName: resourceName}
	needTextLabel := false
	for _, p := range paths {
		switch p {
		case "name":
			label.Name = StringPtr(in.Name)
		case "text_label.background_color", "text_label.description":
			needTextLabel = true
		}
	}
	if needTextLabel {
		label.TextLabel = &common.TextLabel{}
		if in.BackgroundColor != "" {
			label.TextLabel.BackgroundColor = StringPtr(in.BackgroundColor)
		}
		if in.Description != "" {
			label.TextLabel.Description = StringPtr(in.Description)
		}
	}
	_, err = cl.MutateLabels(ctx, &services.MutateLabelsRequest{
		CustomerId: customerID,
		Operations: []*services.LabelOperation{{
			UpdateMask: &fieldmaskpb.FieldMask{Paths: paths},
			Operation:  &services.LabelOperation_Update{Update: label},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) GetLabel(ctx context.Context, resourceName string) (*LabelView, error) {
	customerID, _, err := ParseResourceName(resourceName, "labels")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			label.resource_name,
			label.name,
			label.status,
			label.text_label.background_color,
			label.text_label.description
		FROM label
		WHERE label.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	x := row.GetLabel()
	return &LabelView{
		ResourceName:    x.GetResourceName(),
		Name:            x.GetName(),
		Status:          x.GetStatus().String(),
		BackgroundColor: x.GetTextLabel().GetBackgroundColor(),
		Description:     x.GetTextLabel().GetDescription(),
	}, nil
}

func (c *Client) RemoveLabel(ctx context.Context, resourceName string) error {
	cl, err := c.labelClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "labels")
	if err != nil {
		return err
	}
	_, err = cl.MutateLabels(ctx, &services.MutateLabelsRequest{
		CustomerId: customerID,
		Operations: []*services.LabelOperation{{
			Operation: &services.LabelOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) labelClient(ctx context.Context) (*pbclients.LabelClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.labels != nil {
		return c.labels, nil
	}
	cl, err := pbclients.NewLabelClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: label client: %w", err)
	}
	c.labels = cl
	return cl, nil
}
