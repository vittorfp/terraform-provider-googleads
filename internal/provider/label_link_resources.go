package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

// labelLinkResource is the shared shape of the four label-link resources
// (campaign_label, ad_group_label, ad_group_ad_label,
// ad_group_criterion_label). They're all "M:N join, two immutable
// foreign keys, Create + Remove only on the API". This file declares
// each one as a thin struct around per-resource closures, keeping the
// total surface area small.

type labelLinkResource struct {
	client *googleads.Client

	typeName    string // e.g. "_campaign_label"
	parentField string // e.g. "campaign_id"
	parentDesc  string
	kind        string // e.g. "campaignLabels" — the slash segment

	create func(ctx context.Context, c *googleads.Client, customerID, parent, label string) (string, error)
	get    func(ctx context.Context, c *googleads.Client, resourceName string) (parent string, label string, err error)
	remove func(ctx context.Context, c *googleads.Client, resourceName string) error
}

type labelLinkModel struct {
	ID         types.String `tfsdk:"id"`
	CustomerID types.String `tfsdk:"customer_id"`
	Parent     types.String `tfsdk:"parent"`
	LabelID    types.String `tfsdk:"label_id"`
}

func (r *labelLinkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.typeName
}

func (r *labelLinkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	immutable := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Attaches a `googleads_label` to a " + r.parentDesc + ". Both endpoints are immutable per the API; the link has no update operation, so any change forces resource replacement.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Resource name of the link.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"customer_id": schema.StringAttribute{
				Required: true, Description: "Google Ads customer ID.",
				PlanModifiers: immutable,
			},
			"parent": schema.StringAttribute{
				Required: true, Description: "Resource name of the " + r.parentDesc + ". Immutable.",
				PlanModifiers: immutable,
			},
			"label_id": schema.StringAttribute{
				Required: true, Description: "Resource name of the label. Immutable.",
				PlanModifiers: immutable,
			},
		},
	}
}

func (r *labelLinkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *labelLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan labelLinkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rn, err := r.create(ctx, r.client, plan.CustomerID.ValueString(), plan.Parent.ValueString(), plan.LabelID.ValueString())
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create "+r.typeName, nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	if err := r.refresh(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to read back "+r.typeName, err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *labelLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state labelLinkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.refresh(ctx, &state); err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read "+r.typeName, err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *labelLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan labelLinkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *labelLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state labelLinkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.remove(ctx, r.client, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to remove "+r.typeName, err.Error())
	}
}

func (r *labelLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *labelLinkResource) refresh(ctx context.Context, m *labelLinkModel) error {
	parent, label, err := r.get(ctx, r.client, m.ID.ValueString())
	if err != nil {
		return err
	}
	if cid, _, err := googleads.ParseResourceName(m.ID.ValueString(), r.kind); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.Parent = types.StringValue(parent)
	m.LabelID = types.StringValue(label)
	return nil
}

// Compile-time interface checks for one of the four concrete resources
// (all four embed this struct via constructor functions below).
var (
	_ resource.Resource                   = (*labelLinkResource)(nil)
	_ resource.ResourceWithConfigure      = (*labelLinkResource)(nil)
	_ resource.ResourceWithImportState    = (*labelLinkResource)(nil)
	_ resource.ResourceWithValidateConfig = (*labelLinkResource)(nil)
)

func (r *labelLinkResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg labelLinkModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateSameCustomer(&resp.Diagnostics, cfg.CustomerID, map[string]types.String{
		"parent":   cfg.Parent,
		"label_id": cfg.LabelID,
	})
}

func NewCampaignLabelResource() resource.Resource {
	return &labelLinkResource{
		typeName:    "_campaign_label",
		parentField: "campaign_id",
		parentDesc:  "campaign",
		kind:        "campaignLabels",
		create: func(ctx context.Context, c *googleads.Client, cid, p, l string) (string, error) {
			return c.CreateCampaignLabel(ctx, cid, p, l)
		},
		get: func(ctx context.Context, c *googleads.Client, rn string) (string, string, error) {
			return c.GetCampaignLabel(ctx, rn)
		},
		remove: func(ctx context.Context, c *googleads.Client, rn string) error {
			return c.RemoveCampaignLabel(ctx, rn)
		},
	}
}

func NewAdGroupLabelResource() resource.Resource {
	return &labelLinkResource{
		typeName:    "_ad_group_label",
		parentField: "ad_group_id",
		parentDesc:  "ad group",
		kind:        "adGroupLabels",
		create: func(ctx context.Context, c *googleads.Client, cid, p, l string) (string, error) {
			return c.CreateAdGroupLabel(ctx, cid, p, l)
		},
		get: func(ctx context.Context, c *googleads.Client, rn string) (string, string, error) {
			return c.GetAdGroupLabel(ctx, rn)
		},
		remove: func(ctx context.Context, c *googleads.Client, rn string) error {
			return c.RemoveAdGroupLabel(ctx, rn)
		},
	}
}

func NewAdGroupAdLabelResource() resource.Resource {
	return &labelLinkResource{
		typeName:    "_ad_group_ad_label",
		parentField: "ad_group_ad_id",
		parentDesc:  "ad group ad",
		kind:        "adGroupAdLabels",
		create: func(ctx context.Context, c *googleads.Client, cid, p, l string) (string, error) {
			return c.CreateAdGroupAdLabel(ctx, cid, p, l)
		},
		get: func(ctx context.Context, c *googleads.Client, rn string) (string, string, error) {
			return c.GetAdGroupAdLabel(ctx, rn)
		},
		remove: func(ctx context.Context, c *googleads.Client, rn string) error {
			return c.RemoveAdGroupAdLabel(ctx, rn)
		},
	}
}

func NewAdGroupCriterionLabelResource() resource.Resource {
	return &labelLinkResource{
		typeName:    "_ad_group_criterion_label",
		parentField: "ad_group_criterion_id",
		parentDesc:  "ad group criterion",
		kind:        "adGroupCriterionLabels",
		create: func(ctx context.Context, c *googleads.Client, cid, p, l string) (string, error) {
			return c.CreateAdGroupCriterionLabel(ctx, cid, p, l)
		},
		get: func(ctx context.Context, c *googleads.Client, rn string) (string, string, error) {
			return c.GetAdGroupCriterionLabel(ctx, rn)
		},
		remove: func(ctx context.Context, c *googleads.Client, rn string) error {
			return c.RemoveAdGroupCriterionLabel(ctx, rn)
		},
	}
}
