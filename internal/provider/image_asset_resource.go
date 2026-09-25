package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

var (
	_ resource.Resource                = (*imageAssetResource)(nil)
	_ resource.ResourceWithConfigure   = (*imageAssetResource)(nil)
	_ resource.ResourceWithImportState = (*imageAssetResource)(nil)
)

func NewImageAssetResource() resource.Resource { return &imageAssetResource{} }

type imageAssetResource struct {
	client *googleads.Client
}

type imageAssetModel struct {
	ID          types.String `tfsdk:"id"`
	CustomerID  types.String `tfsdk:"customer_id"`
	Name        types.String `tfsdk:"name"`
	Path        types.String `tfsdk:"path"`
	ContentHash types.String `tfsdk:"content_hash"`
	MimeType    types.String `tfsdk:"mime_type"`
	FileSize    types.Int64  `tfsdk:"file_size"`
}

func (r *imageAssetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image_asset"
}

func (r *imageAssetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An image asset — a PNG, JPEG, or GIF backing image creatives on Performance Max, Display, and other campaign types. The image bytes are read from a local file at apply time; the API never returns the bytes back, so drift detection compares the on-disk file's SHA-256 against the recorded `content_hash`. Asset content is immutable per the API; replacing it requires a new asset. **The API has no remove operation** — Terraform refuses to delete this resource. To stop managing it, unlink any references, remove the block from HCL, and run `terraform state rm`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Asset resource name (customers/{cid}/assets/{id}).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Required: true, Description: "Google Ads customer ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "Optional display name. The API auto-generates one if omitted.",
			},
			"path": schema.StringAttribute{
				Required:    true,
				Description: "Path to a local PNG, JPEG, or GIF. Read at apply time. Changing the path or the file contents forces resource replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"content_hash": schema.StringAttribute{
				Computed:    true,
				Description: "SHA-256 of the file at create time. Plan-time drift in the on-disk file shows here.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mime_type": schema.StringAttribute{
				Computed:    true,
				Description: "Detected MIME type (IMAGE_PNG, IMAGE_JPEG, IMAGE_GIF).",
			},
			"file_size": schema.Int64Attribute{
				Computed:    true,
				Description: "Bytes.",
			},
		},
	}
}

func (r *imageAssetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (r *imageAssetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan imageAssetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data, err := os.ReadFile(plan.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Cannot read image file", err.Error())
		return
	}
	rn, err := r.client.CreateImageAsset(ctx, googleads.AssetInput{
		CustomerID: plan.CustomerID.ValueString(),
		Name:       plan.Name.ValueString(),
		ImageData:  data,
	})
	if err != nil {
		addAPIErrorDiagnostics(&resp.Diagnostics, "Failed to create image asset", nil, err)
		return
	}
	plan.ID = types.StringValue(rn)
	plan.ContentHash = types.StringValue(hashBytes(data))
	view, err := r.client.GetAsset(ctx, rn)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back image asset", err.Error())
		return
	}
	applyImageAssetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *imageAssetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state imageAssetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := r.client.GetAsset(ctx, state.ID.ValueString())
	if err != nil {
		if googleads.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read image asset", err.Error())
		return
	}
	applyImageAssetView(&state, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *imageAssetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state imageAssetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.Name.Equal(state.Name) {
		if err := r.client.UpdateAsset(ctx, state.ID.ValueString(), plan.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to rename image asset", err.Error())
			return
		}
	}
	view, err := r.client.GetAsset(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back image asset", err.Error())
		return
	}
	plan.ID = state.ID
	plan.ContentHash = state.ContentHash
	applyImageAssetView(&plan, view)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *imageAssetResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError(
		"Cannot delete googleads_image_asset",
		"The Google Ads API does not support deleting assets. To stop managing this asset with Terraform: "+
			"(1) unlink it from any ad groups / asset groups that reference it, "+
			"(2) remove it from your HCL, "+
			"(3) run `terraform state rm <address>` to drop it from state. "+
			"The asset itself will remain in the Google Ads account (orphaned).",
	)
}

func (r *imageAssetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyImageAssetView(m *imageAssetModel, v *googleads.AssetView) {
	if cid, _, err := googleads.ParseResourceName(v.ResourceName, "assets"); err == nil {
		m.CustomerID = types.StringValue(cid)
	}
	m.Name = types.StringValue(v.Name)
	m.MimeType = types.StringValue(v.ImageMimeType)
	m.FileSize = types.Int64Value(v.ImageFileSize)
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
