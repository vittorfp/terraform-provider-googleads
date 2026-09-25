package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

var (
	_ resource.Resource                = (*conversionActionResource)(nil)
	_ resource.ResourceWithConfigure   = (*conversionActionResource)(nil)
	_ resource.ResourceWithImportState = (*conversionActionResource)(nil)
)

func NewConversionActionResource() resource.Resource { return &conversionActionResource{} }

type conversionActionResource struct {
	client *googleads.Client
}

type conversionActionModel struct {
	ID                             types.String  `tfsdk:"id"`
	CustomerID                     types.String  `tfsdk:"customer_id"`
	Name                           types.String  `tfsdk:"name"`
	Status                         types.String  `tfsdk:"status"`
	Type                           types.String  `tfsdk:"type"`
	Category                       types.String  `tfsdk:"category"`
	CountingType                   types.String  `tfsdk:"counting_type"`
	ClickThroughLookbackWindowDays types.Int64   `tfsdk:"click_through_lookback_window_days"`
	ViewThroughLookbackWindowDays  types.Int64   `tfsdk:"view_through_lookback_window_days"`
	PrimaryForGoal                 types.Bool    `tfsdk:"primary_for_goal"`
	IncludeInConversionsMetric     types.Bool    `tfsdk:"include_in_conversions_metric"`
	DefaultValue                   types.Float64 `tfsdk:"default_value"`
	DefaultCurrencyCode            types.String  `tfsdk:"default_currency_code"`
	AlwaysUseDefaultValue          types.Bool    `tfsdk:"always_use_default_value"`
}

func (r *conversionActionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_conversion_action"
}

func (r *conversionActionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A conversion action — defines what counts as a conversion (purchase, signup, page view, app install, etc.) so smart bidding strategies (TARGET_CPA, TARGET_ROAS, MAXIMIZE_CONVERSIONS) have something concrete to optimize toward. `type` is immutable per the API; everything else can be updated in place.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource name (customers/{cid}/conversionActions/{id}).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Required: true, Description: "Google Ads customer ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Conversion action name. Must be unique within the account.",
			},
			"status": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "ENABLED (default), REMOVED, or HIDDEN.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Type of conversion action. Common: WEBPAGE, UPLOAD_CLICKS, UPLOAD_CALLS. Immutable.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"category": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Reporting category. Common: DEFAULT, PURCHASE, SIGNUP, PAGE_VIEW, ADD_TO_CART, BEGIN_CHECKOUT, DOWNLOAD, SUBSCRIBE_PAID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"counting_type": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "ONE_PER_CLICK (default for most categories) or MANY_PER_CLICK.",
				Validators: []validator.String{
					stringvalidator.OneOf("ONE_PER_CLICK", "MANY_PER_CLICK"),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"click_through_lookback_window_days": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description:   "How many days a click can precede a conversion event and still count.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"view_through_lookback_window_days": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description:   "How many days an impression can precede a conversion event and still count without an interaction.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"primary_for_goal": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "When true (default), this conversion action drives smart-bidding optimization. When false, it's reported but not biddable.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"include_in_conversions_metric": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "Whether this conversion action contributes to the 'Conversions' column in reports.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"default_value": schema.Float64Attribute{
				Optional: true, Computed: true,
				Description:   "Default value attributed to conversions reported without a value (or when always_use_default_value is true).",
				PlanModifiers: []planmodifier.Float64{float64planmodifier.UseStateForUnknown()},
			},
			"default_currency_code": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "ISO 4217 currency code (e.g. 'USD', 'BRL') for default_value.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"always_use_default_value": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "When true, conversion event values are ignored and default_value is always used.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *conversionActionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *conversionActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan conversionActionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := buildConversionActionInput(&plan)
	rn, err := r.client.CreateConversionAction(ctx, in)
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create conversion action", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	view, err := r.client.GetConversionAction(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back conversion action", err.Error())
		return
	}
	applyConversionActionView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *conversionActionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state conversionActionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetConversionAction(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read conversion action", err.Error())
		return
	}
	applyConversionActionView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *conversionActionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state conversionActionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var paths []string
	if !plan.Name.Equal(state.Name) {
		paths = append(paths, "name")
	}
	if !plan.Status.Equal(state.Status) {
		paths = append(paths, "status")
	}
	if !plan.Category.Equal(state.Category) {
		paths = append(paths, "category")
	}
	if !plan.CountingType.Equal(state.CountingType) {
		paths = append(paths, "counting_type")
	}
	if !plan.ClickThroughLookbackWindowDays.Equal(state.ClickThroughLookbackWindowDays) {
		paths = append(paths, "click_through_lookback_window_days")
	}
	if !plan.ViewThroughLookbackWindowDays.Equal(state.ViewThroughLookbackWindowDays) {
		paths = append(paths, "view_through_lookback_window_days")
	}
	if !plan.PrimaryForGoal.Equal(state.PrimaryForGoal) {
		paths = append(paths, "primary_for_goal")
	}
	if !plan.IncludeInConversionsMetric.Equal(state.IncludeInConversionsMetric) {
		paths = append(paths, "include_in_conversions_metric")
	}
	if !plan.DefaultValue.Equal(state.DefaultValue) {
		paths = append(paths, "value_settings.default_value")
	}
	if !plan.DefaultCurrencyCode.Equal(state.DefaultCurrencyCode) {
		paths = append(paths, "value_settings.default_currency_code")
	}
	if !plan.AlwaysUseDefaultValue.Equal(state.AlwaysUseDefaultValue) {
		paths = append(paths, "value_settings.always_use_default_value")
	}

	if len(paths) > 0 {
		in := buildConversionActionInput(&plan)
		if err := r.client.UpdateConversionAction(ctx, state.ID.ValueString(), in, paths); err != nil {
			addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to update conversion action", nil, err)
			return
		}
	}
	view, err := r.client.GetConversionAction(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back conversion action", err.Error())
		return
	}
	plan.ID = state.ID
	applyConversionActionView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *conversionActionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state conversionActionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveConversionAction(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove conversion action", err.Error())
	}
}

func (r *conversionActionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyConversionActionView(m *conversionActionModel, v *googleads.ConversionActionView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "conversionActions"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.Name = types.StringValue(v.Name)
	m.Status = types.StringValue(v.Status)
	m.Type = types.StringValue(v.Type)
	m.Category = types.StringValue(v.Category)
	m.CountingType = types.StringValue(v.CountingType)
	m.ClickThroughLookbackWindowDays = types.Int64Value(v.ClickThroughLookbackWindowDays)
	m.ViewThroughLookbackWindowDays = types.Int64Value(v.ViewThroughLookbackWindowDays)
	m.PrimaryForGoal = types.BoolValue(v.PrimaryForGoal)
	m.IncludeInConversionsMetric = types.BoolValue(v.IncludeInConversionsMetric)
	m.DefaultValue = types.Float64Value(v.DefaultValue)
	if v.DefaultCurrencyCode == "" {
		m.DefaultCurrencyCode = types.StringNull()
	} else {
		m.DefaultCurrencyCode = types.StringValue(v.DefaultCurrencyCode)
	}
	m.AlwaysUseDefaultValue = types.BoolValue(v.AlwaysUseDefaultValue)
}

func buildConversionActionInput(m *conversionActionModel) googleads.ConversionActionInput {
	in := googleads.ConversionActionInput{
		CustomerID:          m.CustomerID.ValueString(),
		Name:                m.Name.ValueString(),
		Status:              m.Status.ValueString(),
		Type:                m.Type.ValueString(),
		Category:            m.Category.ValueString(),
		CountingType:        m.CountingType.ValueString(),
		DefaultCurrencyCode: m.DefaultCurrencyCode.ValueString(),
	}
	if !m.ClickThroughLookbackWindowDays.IsNull() && !m.ClickThroughLookbackWindowDays.IsUnknown() {
		v := m.ClickThroughLookbackWindowDays.ValueInt64()
		in.ClickThroughLookbackWindowDays = &v
	}
	if !m.ViewThroughLookbackWindowDays.IsNull() && !m.ViewThroughLookbackWindowDays.IsUnknown() {
		v := m.ViewThroughLookbackWindowDays.ValueInt64()
		in.ViewThroughLookbackWindowDays = &v
	}
	if !m.PrimaryForGoal.IsNull() && !m.PrimaryForGoal.IsUnknown() {
		v := m.PrimaryForGoal.ValueBool()
		in.PrimaryForGoal = &v
	}
	if !m.IncludeInConversionsMetric.IsNull() && !m.IncludeInConversionsMetric.IsUnknown() {
		v := m.IncludeInConversionsMetric.ValueBool()
		in.IncludeInConversionsMetric = &v
	}
	if !m.DefaultValue.IsNull() && !m.DefaultValue.IsUnknown() {
		v := m.DefaultValue.ValueFloat64()
		in.DefaultValue = &v
	}
	if !m.AlwaysUseDefaultValue.IsNull() && !m.AlwaysUseDefaultValue.IsUnknown() {
		v := m.AlwaysUseDefaultValue.ValueBool()
		in.AlwaysUseDefaultValue = &v
	}
	return in
}
