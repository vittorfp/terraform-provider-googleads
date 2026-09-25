package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

var (
	_ resource.Resource                     = (*customerNegativeCriterionResource)(nil)
	_ resource.ResourceWithConfigure        = (*customerNegativeCriterionResource)(nil)
	_ resource.ResourceWithImportState      = (*customerNegativeCriterionResource)(nil)
	_ resource.ResourceWithConfigValidators = (*customerNegativeCriterionResource)(nil)
)

func NewCustomerNegativeCriterionResource() resource.Resource {
	return &customerNegativeCriterionResource{}
}

type customerNegativeCriterionResource struct {
	client *googleads.Client
}

type customerNegativeCriterionModel struct {
	ID                    types.String `tfsdk:"id"`
	CustomerID            types.String `tfsdk:"customer_id"`
	Type                  types.String `tfsdk:"type"`
	PlacementURL          types.String `tfsdk:"placement_url"`
	YoutubeChannelID      types.String `tfsdk:"youtube_channel_id"`
	MobileApplicationID   types.String `tfsdk:"mobile_application_id"`
	IPAddress             types.String `tfsdk:"ip_address"`
	NegativeKeywordListID types.String `tfsdk:"negative_keyword_list_id"`
	ContentLabelType      types.String `tfsdk:"content_label_type"`
}

// contentLabelTypeValues is the canonical set of ContentLabelType enum
// values from google.ads.googleads.vN.enums.ContentLabelTypeEnum. Kept
// in sync with the Ads API; we exclude UNSPECIFIED/UNKNOWN since those
// have no business in a config.
var contentLabelTypeValues = []string{
	"SEXUALLY_SUGGESTIVE",
	"BELOW_THE_FOLD",
	"PARKED_DOMAIN",
	"JUVENILE",
	"PROFANITY",
	"TRAGEDY",
	"VIDEO",
	"VIDEO_RATING_DV_G",
	"VIDEO_RATING_DV_PG",
	"VIDEO_RATING_DV_T",
	"VIDEO_RATING_DV_MA",
	"VIDEO_NOT_YET_RATED",
	"EMBEDDED_VIDEO",
	"LIVE_STREAMING_VIDEO",
	"SOCIAL_ISSUES",
	"BRAND_SUITABILITY_CONTENT_FOR_FAMILIES",
	"BRAND_SUITABILITY_GAMES_FIGHTING",
	"BRAND_SUITABILITY_GAMES_MATURE",
	"BRAND_SUITABILITY_HEALTH_SENSITIVE",
	"BRAND_SUITABILITY_HEALTH_SOURCE_UNDETERMINED",
	"BRAND_SUITABILITY_NEWS_RECENT",
	"BRAND_SUITABILITY_NEWS_SENSITIVE",
	"BRAND_SUITABILITY_NEWS_SOURCE_NOT_FEATURED",
	"BRAND_SUITABILITY_POLITICS",
	"BRAND_SUITABILITY_RELIGION",
}

func (r *customerNegativeCriterionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_customer_negative_criterion"
}

func (r *customerNegativeCriterionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	immutable := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Account-wide exclusion. Exactly one of the variant fields (placement_url, youtube_channel_id, mobile_application_id, ip_address, negative_keyword_list_id, content_label_type) must be set. The Ads API has no update operation for these — every field is immutable. Six of the nine API variants are covered; mobile-app categories, YouTube videos, and placement lists land later.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource name (customers/{cid}/customerNegativeCriteria/{id}).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Required: true, Description: "Google Ads customer ID.",
				PlanModifiers: immutable,
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "Output-only criterion type. Reflects which variant attribute is set.",
			},
			"placement_url": schema.StringAttribute{
				Optional: true, Description: "Exclude a specific website placement, e.g. \"http://www.badsite.com\".",
				PlanModifiers: immutable,
			},
			"youtube_channel_id": schema.StringAttribute{
				Optional: true, Description: "Exclude a YouTube channel by its channel ID or channel code.",
				PlanModifiers: immutable,
			},
			"mobile_application_id": schema.StringAttribute{
				Optional: true, Description: "Exclude a mobile app. Format: \"{platform}-{native_id}\" — platform is 1 for iOS, 2 for Android (e.g. \"1-476943146\" or \"2-com.example.app\").",
				PlanModifiers: immutable,
			},
			"ip_address": schema.StringAttribute{
				Optional: true, Description: "Exclude an IP address or CIDR block.",
				PlanModifiers: immutable,
			},
			"negative_keyword_list_id": schema.StringAttribute{
				Optional: true, Description: "Attach a NEGATIVE_KEYWORDS shared set at the customer level. Resource name of a googleads_shared_set.",
				PlanModifiers: immutable,
			},
			"content_label_type": schema.StringAttribute{
				Optional: true, Description: "Exclude an entire content label class account-wide (e.g. SEXUALLY_SUGGESTIVE, BELOW_THE_FOLD, PARKED_DOMAIN, JUVENILE, PROFANITY, TRAGEDY, VIDEO, VIDEO_RATING_DV_G/PG/T/MA, VIDEO_NOT_YET_RATED, EMBEDDED_VIDEO, LIVE_STREAMING_VIDEO, SOCIAL_ISSUES, BRAND_SUITABILITY_*). Values are the ContentLabelType enum from the Ads API.",
				PlanModifiers: immutable,
				Validators: []validator.String{
					stringvalidator.OneOf(contentLabelTypeValues...),
				},
			},
		},
	}
}

func (r *customerNegativeCriterionResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("placement_url"),
			path.MatchRoot("youtube_channel_id"),
			path.MatchRoot("mobile_application_id"),
			path.MatchRoot("ip_address"),
			path.MatchRoot("negative_keyword_list_id"),
			path.MatchRoot("content_label_type"),
		),
	}
}

func (r *customerNegativeCriterionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *customerNegativeCriterionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customerNegativeCriterionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rn, err := r.client.CreateCustomerNegativeCriterion(ctx, googleads.CustomerNegativeCriterionInput{
		CustomerID:            plan.CustomerID.ValueString(),
		PlacementURL:          plan.PlacementURL.ValueString(),
		YoutubeChannelID:      plan.YoutubeChannelID.ValueString(),
		MobileApplicationID:   plan.MobileApplicationID.ValueString(),
		IPAddress:             plan.IPAddress.ValueString(),
		NegativeKeywordListID: plan.NegativeKeywordListID.ValueString(),
		ContentLabelType:      plan.ContentLabelType.ValueString(),
	})
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create customer negative criterion", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetCustomerNegativeCriterion(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back customer negative criterion", err.Error())
		return
	}
	applyCustomerNegativeCriterionView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customerNegativeCriterionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customerNegativeCriterionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetCustomerNegativeCriterion(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read customer negative criterion", err.Error())
		return
	}
	applyCustomerNegativeCriterionView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *customerNegativeCriterionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan customerNegativeCriterionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customerNegativeCriterionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customerNegativeCriterionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveCustomerNegativeCriterion(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove customer negative criterion", err.Error())
	}
}

func (r *customerNegativeCriterionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyCustomerNegativeCriterionView(m *customerNegativeCriterionModel, v *googleads.CustomerNegativeCriterionView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "customerNegativeCriteria"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.Type = types.StringValue(v.Type)
	m.PlacementURL = nullableString(v.PlacementURL)
	m.YoutubeChannelID = nullableString(v.YoutubeChannelID)
	m.MobileApplicationID = nullableString(v.MobileApplicationID)
	m.IPAddress = nullableString(v.IPAddress)
	m.NegativeKeywordListID = nullableString(v.NegativeKeywordListID)
	m.ContentLabelType = nullableString(v.ContentLabelType)
}

func nullableString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
