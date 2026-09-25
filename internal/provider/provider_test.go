package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testProviderFactories returns the provider factory map used by every
// acceptance test in this package. The provider is configured purely from
// GOOGLE_ADS_* env vars.
var testProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"googleads": providerserver.NewProtocol6WithError(New("test")()),
}

// requireAccEnv aborts the test unless every credential env var the
// provider needs is set. Acceptance tests cost real Ads API calls — we'd
// rather skip them than fail confusingly when keys are missing.
func requireAccEnv(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 to run acceptance tests")
	}
	required := []string{
		"GOOGLE_ADS_LOGIN_CUSTOMER_ID",
		"GOOGLE_ADS_CLIENT_ID",
		"GOOGLE_ADS_CLIENT_SECRET",
		"GOOGLE_ADS_REFRESH_TOKEN",
		"GOOGLE_ADS_TEST_CUSTOMER_ID",
	}
	for _, k := range required {
		if os.Getenv(k) == "" {
			t.Fatalf("%s must be set for acceptance tests", k)
		}
	}
}

func testCustomerID() string { return os.Getenv("GOOGLE_ADS_TEST_CUSTOMER_ID") }
