package googleads

import "context"

// List* methods enumerate every non-REMOVED resource of the given kind under
// a customer ID. They power the googleads-tfgen code generator.

func (c *Client) ListBudgets(ctx context.Context, customerID string) ([]BudgetView, error) {
	rows, err := c.Search(ctx, customerID, `
		SELECT
			campaign_budget.resource_name,
			campaign_budget.name,
			campaign_budget.amount_micros,
			campaign_budget.delivery_method,
			campaign_budget.explicitly_shared,
			campaign_budget.status
		FROM campaign_budget
		WHERE campaign_budget.status != 'REMOVED'`)
	if err != nil {
		return nil, err
	}
	out := make([]BudgetView, 0, len(rows))
	for _, row := range rows {
		b := row.GetCampaignBudget()
		out = append(out, BudgetView{
			ResourceName:     b.GetResourceName(),
			Name:             b.GetName(),
			AmountMicros:     b.GetAmountMicros(),
			DeliveryMethod:   b.GetDeliveryMethod().String(),
			ExplicitlyShared: b.GetExplicitlyShared(),
			Status:           b.GetStatus().String(),
		})
	}
	return out, nil
}

func (c *Client) ListCampaigns(ctx context.Context, customerID string) ([]CampaignView, error) {
	rows, err := c.Search(ctx, customerID,
		campaignSelectClause+`WHERE campaign.status != 'REMOVED'`)
	if err != nil {
		return nil, err
	}
	out := make([]CampaignView, 0, len(rows))
	for _, row := range rows {
		out = append(out, campaignViewFromRow(row.GetCampaign()))
	}
	return out, nil
}

func (c *Client) ListAdGroups(ctx context.Context, customerID string) ([]AdGroupView, error) {
	rows, err := c.Search(ctx, customerID, `
		SELECT
			ad_group.resource_name,
			ad_group.campaign,
			ad_group.name,
			ad_group.status,
			ad_group.type,
			ad_group.cpc_bid_micros
		FROM ad_group
		WHERE ad_group.status != 'REMOVED'`)
	if err != nil {
		return nil, err
	}
	out := make([]AdGroupView, 0, len(rows))
	for _, row := range rows {
		g := row.GetAdGroup()
		out = append(out, AdGroupView{
			ResourceName: g.GetResourceName(),
			Campaign:     g.GetCampaign(),
			Name:         g.GetName(),
			Status:       g.GetStatus().String(),
			Type:         g.GetType().String(),
			CpcBidMicros: g.GetCpcBidMicros(),
		})
	}
	return out, nil
}

// ListResponsiveSearchAds returns only ad_group_ads whose Ad type is
// RESPONSIVE_SEARCH_AD — the only ad type the provider supports today.
// Non-RSA ads are silently skipped.
func (c *Client) ListResponsiveSearchAds(ctx context.Context, customerID string) ([]AdGroupAdView, error) {
	rows, err := c.Search(ctx, customerID, `
		SELECT
			ad_group_ad.resource_name,
			ad_group_ad.ad_group,
			ad_group_ad.status,
			ad_group_ad.ad.final_urls,
			ad_group_ad.ad.responsive_search_ad.headlines,
			ad_group_ad.ad.responsive_search_ad.descriptions,
			ad_group_ad.ad.responsive_search_ad.path1,
			ad_group_ad.ad.responsive_search_ad.path2
		FROM ad_group_ad
		WHERE ad_group_ad.status != 'REMOVED'
		  AND ad_group_ad.ad.type = 'RESPONSIVE_SEARCH_AD'`)
	if err != nil {
		return nil, err
	}
	out := make([]AdGroupAdView, 0, len(rows))
	for _, row := range rows {
		a := row.GetAdGroupAd()
		rsa := a.GetAd().GetResponsiveSearchAd()
		view := AdGroupAdView{
			ResourceName: a.GetResourceName(),
			AdGroup:      a.GetAdGroup(),
			Status:       a.GetStatus().String(),
			FinalURLs:    a.GetAd().GetFinalUrls(),
			Path1:        rsa.GetPath1(),
			Path2:        rsa.GetPath2(),
		}
		for _, h := range rsa.GetHeadlines() {
			view.Headlines = append(view.Headlines, h.GetText())
		}
		for _, d := range rsa.GetDescriptions() {
			view.Descriptions = append(view.Descriptions, d.GetText())
		}
		out = append(out, view)
	}
	return out, nil
}

// ListKeywords returns ad_group_criteria of type KEYWORD (positive and
// negative). Other criterion types (placements, audiences, age ranges, etc.)
// are silently skipped.
func (c *Client) ListKeywords(ctx context.Context, customerID string) ([]CriterionView, error) {
	rows, err := c.Search(ctx, customerID, `
		SELECT
			ad_group_criterion.resource_name,
			ad_group_criterion.ad_group,
			ad_group_criterion.status,
			ad_group_criterion.negative,
			ad_group_criterion.cpc_bid_micros,
			ad_group_criterion.keyword.text,
			ad_group_criterion.keyword.match_type
		FROM ad_group_criterion
		WHERE ad_group_criterion.status != 'REMOVED'
		  AND ad_group_criterion.type = 'KEYWORD'`)
	if err != nil {
		return nil, err
	}
	out := make([]CriterionView, 0, len(rows))
	for _, row := range rows {
		x := row.GetAdGroupCriterion()
		out = append(out, CriterionView{
			ResourceName: x.GetResourceName(),
			AdGroup:      x.GetAdGroup(),
			Status:       x.GetStatus().String(),
			Negative:     x.GetNegative(),
			CpcBidMicros: x.GetCpcBidMicros(),
			KeywordText:  x.GetKeyword().GetText(),
			MatchType:    x.GetKeyword().GetMatchType().String(),
		})
	}
	return out, nil
}
