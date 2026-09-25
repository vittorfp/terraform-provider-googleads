package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

var (
	_ datasource.DataSource              = (*customerDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*customerDataSource)(nil)
)

func NewCustomerDataSource() datasource.DataSource { return &customerDataSource{} }

type customerDataSource struct {
	client *googleads.Client
}

type customerDataSourceModel struct {
	CustomerID      types.String `tfsdk:"customer_id"`
	ResourceName    types.String `tfsdk:"resource_name"`
	DescriptiveName types.String `tfsdk:"descriptive_name"`
	CurrencyCode    types.String `tfsdk:"currency_code"`
	TimeZone        types.String `tfsdk:"time_zone"`
	Manager         types.Bool   `tfsdk:"manager"`
	TestAccount     types.Bool   `tfsdk:"test_account"`
}

func (d *customerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_customer"
}

func (d *customerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Read-only metadata about a Google Ads customer (account). Useful for guarding HCL with preconditions on test_account / manager.",
		Attributes: map[string]schema.Attribute{
			"customer_id": schema.StringAttribute{
				Required:    true,
				Description: "Google Ads customer ID (digits only).",
			},
			"resource_name":    schema.StringAttribute{Computed: true, Description: "customers/{id}"},
			"descriptive_name": schema.StringAttribute{Computed: true, Description: "Account display name."},
			"currency_code":    schema.StringAttribute{Computed: true, Description: "ISO 4217 currency, e.g. USD."},
			"time_zone":        schema.StringAttribute{Computed: true, Description: "IANA time zone."},
			"manager":          schema.BoolAttribute{Computed: true, Description: "True if this is an MCC (manager) account."},
			"test_account":     schema.BoolAttribute{Computed: true, Description: "True if this is a sandbox/test account."},
		},
	}
}

func (d *customerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerClient(req.ProviderData, &resp.Diagnostics)
}

func (d *customerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state customerDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	view, err := d.client.GetCustomer(ctx, state.CustomerID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read customer", err.Error())
		return
	}
	state.ResourceName = types.StringValue(view.ResourceName)
	state.DescriptiveName = types.StringValue(view.DescriptiveName)
	state.CurrencyCode = types.StringValue(view.CurrencyCode)
	state.TimeZone = types.StringValue(view.TimeZone)
	state.Manager = types.BoolValue(view.Manager)
	state.TestAccount = types.BoolValue(view.TestAccount)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
