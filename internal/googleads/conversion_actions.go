package googleads

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pbclients "github.com/shenzhencenter/google-ads-pb/clients"
	"github.com/shenzhencenter/google-ads-pb/enums"
	"github.com/shenzhencenter/google-ads-pb/resources"
	"github.com/shenzhencenter/google-ads-pb/services"
)

// ConversionActionInput models the practical subset of conversion-action
// fields most users need. Out of scope for v1: app/Firebase/GA4 settings,
// phone-call duration, attribution model overrides — they have their own
// sub-objects and validation surface and are better handled separately
// once a real user needs them.
type ConversionActionInput struct {
	CustomerID                     string
	Name                           string
	Status                         string // ENABLED | REMOVED | HIDDEN
	Type                           string // WEBPAGE | UPLOAD_CLICKS | UPLOAD_CALLS | ... (immutable)
	Category                       string // DEFAULT | PURCHASE | SIGNUP | PAGE_VIEW | ...
	CountingType                   string // ONE_PER_CLICK | MANY_PER_CLICK
	ClickThroughLookbackWindowDays *int64
	ViewThroughLookbackWindowDays  *int64
	PrimaryForGoal                 *bool
	IncludeInConversionsMetric     *bool

	DefaultValue          *float64
	DefaultCurrencyCode   string
	AlwaysUseDefaultValue *bool
}

type ConversionActionView struct {
	ResourceName                   string
	Name                           string
	Status                         string
	Type                           string
	Category                       string
	CountingType                   string
	ClickThroughLookbackWindowDays int64
	ViewThroughLookbackWindowDays  int64
	PrimaryForGoal                 bool
	IncludeInConversionsMetric     bool

	DefaultValue          float64
	DefaultCurrencyCode   string
	AlwaysUseDefaultValue bool
}

func (c *Client) CreateConversionAction(ctx context.Context, in ConversionActionInput) (string, error) {
	cl, err := c.conversionActionClient(ctx)
	if err != nil {
		return "", err
	}
	ca := buildConversionAction(in)
	resp, err := cl.MutateConversionActions(ctx, &services.MutateConversionActionsRequest{
		CustomerId: in.CustomerID,
		Operations: []*services.ConversionActionOperation{{
			Operation: &services.ConversionActionOperation_Create{Create: ca},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

// UpdateConversionAction applies field-mask paths. `type` is immutable per
// the API; passing it in paths will be rejected.
func (c *Client) UpdateConversionAction(ctx context.Context, resourceName string, in ConversionActionInput, paths []string) error {
	cl, err := c.conversionActionClient(ctx)
	if err != nil {
		return err
	}
	ca := buildConversionAction(in)
	ca.ResourceName = resourceName
	customerID, _, err := ParseResourceName(resourceName, "conversionActions")
	if err != nil {
		return err
	}
	_, err = cl.MutateConversionActions(ctx, &services.MutateConversionActionsRequest{
		CustomerId: customerID,
		Operations: []*services.ConversionActionOperation{{
			UpdateMask: &fieldmaskpb.FieldMask{Paths: paths},
			Operation:  &services.ConversionActionOperation_Update{Update: ca},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) GetConversionAction(ctx context.Context, resourceName string) (*ConversionActionView, error) {
	customerID, _, err := ParseResourceName(resourceName, "conversionActions")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			conversion_action.resource_name,
			conversion_action.name,
			conversion_action.status,
			conversion_action.type,
			conversion_action.category,
			conversion_action.counting_type,
			conversion_action.click_through_lookback_window_days,
			conversion_action.view_through_lookback_window_days,
			conversion_action.primary_for_goal,
			conversion_action.include_in_conversions_metric,
			conversion_action.value_settings.default_value,
			conversion_action.value_settings.default_currency_code,
			conversion_action.value_settings.always_use_default_value
		FROM conversion_action
		WHERE conversion_action.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	x := row.GetConversionAction()
	vs := x.GetValueSettings()
	return &ConversionActionView{
		ResourceName:                   x.GetResourceName(),
		Name:                           x.GetName(),
		Status:                         x.GetStatus().String(),
		Type:                           x.GetType().String(),
		Category:                       x.GetCategory().String(),
		CountingType:                   x.GetCountingType().String(),
		ClickThroughLookbackWindowDays: x.GetClickThroughLookbackWindowDays(),
		ViewThroughLookbackWindowDays:  x.GetViewThroughLookbackWindowDays(),
		PrimaryForGoal:                 x.GetPrimaryForGoal(),
		IncludeInConversionsMetric:     x.GetIncludeInConversionsMetric(),
		DefaultValue:                   vs.GetDefaultValue(),
		DefaultCurrencyCode:            vs.GetDefaultCurrencyCode(),
		AlwaysUseDefaultValue:          vs.GetAlwaysUseDefaultValue(),
	}, nil
}

func (c *Client) RemoveConversionAction(ctx context.Context, resourceName string) error {
	cl, err := c.conversionActionClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "conversionActions")
	if err != nil {
		return err
	}
	_, err = cl.MutateConversionActions(ctx, &services.MutateConversionActionsRequest{
		CustomerId: customerID,
		Operations: []*services.ConversionActionOperation{{
			Operation: &services.ConversionActionOperation_Remove{Remove: resourceName},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) conversionActionClient(ctx context.Context) (*pbclients.ConversionActionClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conversionActions != nil {
		return c.conversionActions, nil
	}
	cl, err := pbclients.NewConversionActionClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: conversion action client: %w", err)
	}
	c.conversionActions = cl
	return cl, nil
}

func buildConversionAction(in ConversionActionInput) *resources.ConversionAction {
	ca := &resources.ConversionAction{
		Name: StringPtr(in.Name),
	}
	if in.Status != "" {
		ca.Status = enums.ConversionActionStatusEnum_ConversionActionStatus(enums.ConversionActionStatusEnum_ConversionActionStatus_value[in.Status])
	}
	if in.Type != "" {
		ca.Type = enums.ConversionActionTypeEnum_ConversionActionType(enums.ConversionActionTypeEnum_ConversionActionType_value[in.Type])
	}
	if in.Category != "" {
		ca.Category = enums.ConversionActionCategoryEnum_ConversionActionCategory(enums.ConversionActionCategoryEnum_ConversionActionCategory_value[in.Category])
	}
	if in.CountingType != "" {
		ca.CountingType = enums.ConversionActionCountingTypeEnum_ConversionActionCountingType(enums.ConversionActionCountingTypeEnum_ConversionActionCountingType_value[in.CountingType])
	}
	if in.ClickThroughLookbackWindowDays != nil {
		ca.ClickThroughLookbackWindowDays = in.ClickThroughLookbackWindowDays
	}
	if in.ViewThroughLookbackWindowDays != nil {
		ca.ViewThroughLookbackWindowDays = in.ViewThroughLookbackWindowDays
	}
	if in.PrimaryForGoal != nil {
		ca.PrimaryForGoal = in.PrimaryForGoal
	}
	if in.IncludeInConversionsMetric != nil {
		ca.IncludeInConversionsMetric = in.IncludeInConversionsMetric
	}
	if in.DefaultValue != nil || in.DefaultCurrencyCode != "" || in.AlwaysUseDefaultValue != nil {
		vs := &resources.ConversionAction_ValueSettings{}
		if in.DefaultValue != nil {
			vs.DefaultValue = in.DefaultValue
		}
		if in.DefaultCurrencyCode != "" {
			vs.DefaultCurrencyCode = StringPtr(in.DefaultCurrencyCode)
		}
		if in.AlwaysUseDefaultValue != nil {
			vs.AlwaysUseDefaultValue = in.AlwaysUseDefaultValue
		}
		ca.ValueSettings = vs
	}
	return ca
}
