package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

// providerClient unwraps the *googleads.Client that the provider's Configure
// step stashes in resp.ResourceData / resp.DataSourceData. If the caller is
// invoked before Configure runs (e.g. during plan validation with no
// provider) the typed nil is returned with no diagnostic — let the framework
// surface the issue.
func providerClient(raw any, diags *diag.Diagnostics) *googleads.Client {
	if raw == nil {
		return nil
	}
	client, ok := raw.(*googleads.Client)
	if !ok {
		diags.AddError(
			"Unexpected provider data",
			fmt.Sprintf("Expected *googleads.Client, got %T. This is a bug in terraform-provider-googleads.", raw),
		)
		return nil
	}
	return client
}
