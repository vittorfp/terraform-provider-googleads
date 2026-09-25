package googleads

import "context"

// List* methods for the resource types added after PR #1. The original
// list.go has the budget/campaign/ad-group/RSA/keyword set; this file
// adds everything the generator needs to do a full-account import.

func (c *Client) ListTextAssets(ctx context.Context, customerID string) ([]AssetView, error) {
	rows, err := c.Search(ctx, customerID, `
		SELECT
			asset.resource_name,
			asset.name,
			asset.type,
			asset.text_asset.text
		FROM asset
		WHERE asset.type = 'TEXT'`)
	if err != nil {
		return nil, err
	}
	out := make([]AssetView, 0, len(rows))
	for _, row := range rows {
		a := row.GetAsset()
		out = append(out, AssetView{
			ResourceName: a.GetResourceName(),
			Name:         a.GetName(),
			Type:         a.GetType().String(),
			Text:         a.GetTextAsset().GetText(),
		})
	}
	return out, nil
}

func (c *Client) ListAssetGroups(ctx context.Context, customerID string) ([]AssetGroupView, error) {
	rows, err := c.Search(ctx, customerID, `
		SELECT
			asset_group.resource_name,
			asset_group.campaign,
			asset_group.name,
			asset_group.status,
			asset_group.final_urls,
			asset_group.final_mobile_urls,
			asset_group.path1,
			asset_group.path2
		FROM asset_group
		WHERE asset_group.status != 'REMOVED'`)
	if err != nil {
		return nil, err
	}
	out := make([]AssetGroupView, 0, len(rows))
	for _, row := range rows {
		x := row.GetAssetGroup()
		out = append(out, AssetGroupView{
			ResourceName:    x.GetResourceName(),
			Campaign:        x.GetCampaign(),
			Name:            x.GetName(),
			Status:          x.GetStatus().String(),
			FinalURLs:       x.GetFinalUrls(),
			FinalMobileURLs: x.GetFinalMobileUrls(),
			Path1:           x.GetPath1(),
			Path2:           x.GetPath2(),
		})
	}
	return out, nil
}

func (c *Client) ListAssetGroupAssets(ctx context.Context, customerID string) ([]AssetGroupAssetView, error) {
	rows, err := c.Search(ctx, customerID, `
		SELECT
			asset_group_asset.resource_name,
			asset_group_asset.asset_group,
			asset_group_asset.asset,
			asset_group_asset.field_type,
			asset_group_asset.status
		FROM asset_group_asset
		WHERE asset_group_asset.status != 'REMOVED'`)
	if err != nil {
		return nil, err
	}
	out := make([]AssetGroupAssetView, 0, len(rows))
	for _, row := range rows {
		x := row.GetAssetGroupAsset()
		out = append(out, AssetGroupAssetView{
			ResourceName: x.GetResourceName(),
			AssetGroup:   x.GetAssetGroup(),
			Asset:        x.GetAsset(),
			FieldType:    x.GetFieldType().String(),
			Status:       x.GetStatus().String(),
		})
	}
	return out, nil
}

func (c *Client) ListConversionActions(ctx context.Context, customerID string) ([]ConversionActionView, error) {
	rows, err := c.Search(ctx, customerID, `
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
		WHERE conversion_action.status != 'REMOVED'`)
	if err != nil {
		return nil, err
	}
	out := make([]ConversionActionView, 0, len(rows))
	for _, row := range rows {
		x := row.GetConversionAction()
		vs := x.GetValueSettings()
		out = append(out, ConversionActionView{
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
		})
	}
	return out, nil
}

func (c *Client) ListSharedSets(ctx context.Context, customerID string) ([]SharedSetView, error) {
	rows, err := c.Search(ctx, customerID, `
		SELECT shared_set.resource_name, shared_set.name, shared_set.type, shared_set.status
		FROM shared_set
		WHERE shared_set.status != 'REMOVED'`)
	if err != nil {
		return nil, err
	}
	out := make([]SharedSetView, 0, len(rows))
	for _, row := range rows {
		x := row.GetSharedSet()
		out = append(out, SharedSetView{
			ResourceName: x.GetResourceName(),
			Name:         x.GetName(),
			Type:         x.GetType().String(),
			Status:       x.GetStatus().String(),
		})
	}
	return out, nil
}

func (c *Client) ListSharedCriteria(ctx context.Context, customerID string) ([]SharedCriterionView, error) {
	rows, err := c.Search(ctx, customerID, `
		SELECT
			shared_criterion.resource_name,
			shared_criterion.shared_set,
			shared_criterion.keyword.text,
			shared_criterion.keyword.match_type
		FROM shared_criterion
		WHERE shared_criterion.type = 'KEYWORD'`)
	if err != nil {
		return nil, err
	}
	out := make([]SharedCriterionView, 0, len(rows))
	for _, row := range rows {
		x := row.GetSharedCriterion()
		out = append(out, SharedCriterionView{
			ResourceName: x.GetResourceName(),
			SharedSet:    x.GetSharedSet(),
			KeywordText:  x.GetKeyword().GetText(),
			MatchType:    x.GetKeyword().GetMatchType().String(),
		})
	}
	return out, nil
}

func (c *Client) ListCampaignSharedSets(ctx context.Context, customerID string) ([]CampaignSharedSetView, error) {
	rows, err := c.Search(ctx, customerID, `
		SELECT
			campaign_shared_set.resource_name,
			campaign_shared_set.campaign,
			campaign_shared_set.shared_set,
			campaign_shared_set.status
		FROM campaign_shared_set
		WHERE campaign_shared_set.status != 'REMOVED'`)
	if err != nil {
		return nil, err
	}
	out := make([]CampaignSharedSetView, 0, len(rows))
	for _, row := range rows {
		x := row.GetCampaignSharedSet()
		out = append(out, CampaignSharedSetView{
			ResourceName: x.GetResourceName(),
			Campaign:     x.GetCampaign(),
			SharedSet:    x.GetSharedSet(),
			Status:       x.GetStatus().String(),
		})
	}
	return out, nil
}

func (c *Client) ListCustomerNegativeCriteria(ctx context.Context, customerID string) ([]CustomerNegativeCriterionView, error) {
	rows, err := c.Search(ctx, customerID, `
		SELECT
			customer_negative_criterion.resource_name,
			customer_negative_criterion.type,
			customer_negative_criterion.placement.url,
			customer_negative_criterion.youtube_channel.channel_id,
			customer_negative_criterion.mobile_application.app_id,
			customer_negative_criterion.ip_block.ip_address,
			customer_negative_criterion.negative_keyword_list.shared_set,
			customer_negative_criterion.content_label.type
		FROM customer_negative_criterion`)
	if err != nil {
		return nil, err
	}
	out := make([]CustomerNegativeCriterionView, 0, len(rows))
	for _, row := range rows {
		x := row.GetCustomerNegativeCriterion()
		out = append(out, CustomerNegativeCriterionView{
			ResourceName:          x.GetResourceName(),
			Type:                  x.GetType().String(),
			PlacementURL:          x.GetPlacement().GetUrl(),
			YoutubeChannelID:      x.GetYoutubeChannel().GetChannelId(),
			MobileApplicationID:   x.GetMobileApplication().GetAppId(),
			IPAddress:             x.GetIpBlock().GetIpAddress(),
			NegativeKeywordListID: x.GetNegativeKeywordList().GetSharedSet(),
			ContentLabelType:      contentLabelTypeString(x.GetContentLabel().GetType()),
		})
	}
	return out, nil
}

func (c *Client) ListCampaignCriteria(ctx context.Context, customerID string) ([]CampaignCriterionView, error) {
	rows, err := c.Search(ctx, customerID, `
		SELECT
			campaign_criterion.resource_name,
			campaign_criterion.campaign,
			campaign_criterion.status,
			campaign_criterion.type,
			campaign_criterion.bid_modifier,
			campaign_criterion.negative,
			campaign_criterion.location.geo_target_constant,
			campaign_criterion.language.language_constant,
			campaign_criterion.device.type,
			campaign_criterion.ip_block.ip_address,
			campaign_criterion.ad_schedule.day_of_week,
			campaign_criterion.ad_schedule.start_hour,
			campaign_criterion.ad_schedule.end_hour,
			campaign_criterion.ad_schedule.start_minute,
			campaign_criterion.ad_schedule.end_minute
		FROM campaign_criterion
		WHERE campaign_criterion.status != 'REMOVED'
		  AND campaign_criterion.type IN ('LOCATION', 'LANGUAGE', 'DEVICE', 'AD_SCHEDULE', 'IP_BLOCK')`)
	if err != nil {
		return nil, err
	}
	out := make([]CampaignCriterionView, 0, len(rows))
	for _, row := range rows {
		x := row.GetCampaignCriterion()
		out = append(out, CampaignCriterionView{
			ResourceName:          x.GetResourceName(),
			Campaign:              x.GetCampaign(),
			Status:                x.GetStatus().String(),
			Type:                  x.GetType().String(),
			BidModifier:           x.GetBidModifier(),
			Negative:              x.GetNegative(),
			LocationID:            x.GetLocation().GetGeoTargetConstant(),
			LanguageID:            x.GetLanguage().GetLanguageConstant(),
			DeviceType:            x.GetDevice().GetType().String(),
			IPAddress:             x.GetIpBlock().GetIpAddress(),
			AdScheduleDayOfWeek:   x.GetAdSchedule().GetDayOfWeek().String(),
			AdScheduleStartHour:   x.GetAdSchedule().GetStartHour(),
			AdScheduleEndHour:     x.GetAdSchedule().GetEndHour(),
			AdScheduleStartMinute: x.GetAdSchedule().GetStartMinute().String(),
			AdScheduleEndMinute:   x.GetAdSchedule().GetEndMinute().String(),
		})
	}
	return out, nil
}

func (c *Client) ListAudienceCriteria(ctx context.Context, customerID string) ([]AudienceCriterionView, error) {
	rows, err := c.Search(ctx, customerID, `
		SELECT
			ad_group_criterion.resource_name,
			ad_group_criterion.ad_group,
			ad_group_criterion.status,
			ad_group_criterion.bid_modifier,
			ad_group_criterion.negative,
			ad_group_criterion.user_list.user_list,
			ad_group_criterion.age_range.type,
			ad_group_criterion.gender.type
		FROM ad_group_criterion
		WHERE ad_group_criterion.status != 'REMOVED'
		  AND ad_group_criterion.type IN ('USER_LIST', 'AGE_RANGE', 'GENDER')`)
	if err != nil {
		return nil, err
	}
	out := make([]AudienceCriterionView, 0, len(rows))
	for _, row := range rows {
		x := row.GetAdGroupCriterion()
		out = append(out, AudienceCriterionView{
			ResourceName: x.GetResourceName(),
			AdGroup:      x.GetAdGroup(),
			Status:       x.GetStatus().String(),
			BidModifier:  x.GetBidModifier(),
			Negative:     x.GetNegative(),
			UserListID:   x.GetUserList().GetUserList(),
			AgeRangeType: x.GetAgeRange().GetType().String(),
			GenderType:   x.GetGender().GetType().String(),
		})
	}
	return out, nil
}

func (c *Client) ListLabels(ctx context.Context, customerID string) ([]LabelView, error) {
	rows, err := c.Search(ctx, customerID, `
		SELECT
			label.resource_name,
			label.name,
			label.status,
			label.text_label.background_color,
			label.text_label.description
		FROM label
		WHERE label.status != 'REMOVED'`)
	if err != nil {
		return nil, err
	}
	out := make([]LabelView, 0, len(rows))
	for _, row := range rows {
		x := row.GetLabel()
		out = append(out, LabelView{
			ResourceName:    x.GetResourceName(),
			Name:            x.GetName(),
			Status:          x.GetStatus().String(),
			BackgroundColor: x.GetTextLabel().GetBackgroundColor(),
			Description:     x.GetTextLabel().GetDescription(),
		})
	}
	return out, nil
}

// LabelLinkView is the shared shape of the four *_label join rows for
// generator-side iteration. Each row carries (resource_name, parent
// resource name, label resource name).
type LabelLinkView struct {
	ResourceName string
	Parent       string
	Label        string
}

func (c *Client) ListCampaignLabels(ctx context.Context, customerID string) ([]LabelLinkView, error) {
	rows, err := c.Search(ctx, customerID, `
		SELECT campaign_label.resource_name, campaign_label.campaign, campaign_label.label
		FROM campaign_label`)
	if err != nil {
		return nil, err
	}
	out := make([]LabelLinkView, 0, len(rows))
	for _, row := range rows {
		x := row.GetCampaignLabel()
		out = append(out, LabelLinkView{ResourceName: x.GetResourceName(), Parent: x.GetCampaign(), Label: x.GetLabel()})
	}
	return out, nil
}

func (c *Client) ListAdGroupLabels(ctx context.Context, customerID string) ([]LabelLinkView, error) {
	rows, err := c.Search(ctx, customerID, `
		SELECT ad_group_label.resource_name, ad_group_label.ad_group, ad_group_label.label
		FROM ad_group_label`)
	if err != nil {
		return nil, err
	}
	out := make([]LabelLinkView, 0, len(rows))
	for _, row := range rows {
		x := row.GetAdGroupLabel()
		out = append(out, LabelLinkView{ResourceName: x.GetResourceName(), Parent: x.GetAdGroup(), Label: x.GetLabel()})
	}
	return out, nil
}

func (c *Client) ListAdGroupAdLabels(ctx context.Context, customerID string) ([]LabelLinkView, error) {
	rows, err := c.Search(ctx, customerID, `
		SELECT ad_group_ad_label.resource_name, ad_group_ad_label.ad_group_ad, ad_group_ad_label.label
		FROM ad_group_ad_label`)
	if err != nil {
		return nil, err
	}
	out := make([]LabelLinkView, 0, len(rows))
	for _, row := range rows {
		x := row.GetAdGroupAdLabel()
		out = append(out, LabelLinkView{ResourceName: x.GetResourceName(), Parent: x.GetAdGroupAd(), Label: x.GetLabel()})
	}
	return out, nil
}

func (c *Client) ListAdGroupCriterionLabels(ctx context.Context, customerID string) ([]LabelLinkView, error) {
	rows, err := c.Search(ctx, customerID, `
		SELECT ad_group_criterion_label.resource_name, ad_group_criterion_label.ad_group_criterion, ad_group_criterion_label.label
		FROM ad_group_criterion_label`)
	if err != nil {
		return nil, err
	}
	out := make([]LabelLinkView, 0, len(rows))
	for _, row := range rows {
		x := row.GetAdGroupCriterionLabel()
		out = append(out, LabelLinkView{ResourceName: x.GetResourceName(), Parent: x.GetAdGroupCriterion(), Label: x.GetLabel()})
	}
	return out, nil
}
