// Command googleads-diag tests the configured credentials against an MCC
// and one or more child customers, printing what the API actually returns
// at each step.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/vittorfp/terraform-provider-googleads/internal/googleads"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: googleads-diag <customer_id> [<customer_id>...]")
	}
	cfg := googleads.Config{
		DeveloperToken:  os.Getenv("GOOGLE_ADS_DEVELOPER_TOKEN"),
		LoginCustomerID: os.Getenv("GOOGLE_ADS_LOGIN_CUSTOMER_ID"),
		ClientID:        os.Getenv("GOOGLE_ADS_CLIENT_ID"),
		ClientSecret:    os.Getenv("GOOGLE_ADS_CLIENT_SECRET"),
		RefreshToken:    os.Getenv("GOOGLE_ADS_REFRESH_TOKEN"),
	}
	if cfg.RefreshToken == "" {
		if path, err := googleads.DefaultCachePath(); err == nil {
			if rt, _ := googleads.LoadCachedRefreshToken(path); rt != "" {
				cfg.RefreshToken = rt
			}
		}
	}

	ctx := context.Background()
	client, err := googleads.NewClient(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fmt.Println("MCC (login_customer_id):", cfg.LoginCustomerID)

	fmt.Println("\n[0] Listing customers directly accessible to the OAuth user:")
	accessible, err := client.ListAccessibleCustomers(ctx)
	if err != nil {
		fmt.Println("  ERROR:", err)
	} else if len(accessible) == 0 {
		fmt.Println("  (none — the Google account you signed in with has no Ads access)")
	} else {
		for _, rn := range accessible {
			fmt.Println("  -", rn)
		}
	}

	fmt.Println("\n[1] Reading the MCC's own customer record:")
	if c, err := client.GetCustomer(ctx, cfg.LoginCustomerID); err != nil {
		fmt.Println("  ERROR:", err)
	} else {
		fmt.Printf("  OK: id=%d name=%q currency=%s manager=%v test=%v\n",
			c.ID, c.DescriptiveName, c.CurrencyCode, c.Manager, c.TestAccount)
	}

	fmt.Println("\n[2] Listing direct-child customers of the MCC:")
	rows, err := client.Search(ctx, cfg.LoginCustomerID, `
		SELECT
			customer_client.id,
			customer_client.descriptive_name,
			customer_client.manager,
			customer_client.test_account,
			customer_client.level
		FROM customer_client
		WHERE customer_client.level <= 1`)
	if err != nil {
		fmt.Println("  ERROR:", err)
	} else if len(rows) == 0 {
		fmt.Println("  (no children — is this actually an MCC?)")
	} else {
		for _, r := range rows {
			cc := r.GetCustomerClient()
			fmt.Printf("  - id=%d name=%q manager=%v test=%v level=%d\n",
				cc.GetId(), cc.GetDescriptiveName(), cc.GetManager(), cc.GetTestAccount(), cc.GetLevel())
		}
	}

	for _, cid := range os.Args[1:] {
		fmt.Printf("\n[3] Reading target customer %s directly:\n", cid)
		if c, err := client.GetCustomer(ctx, cid); err != nil {
			fmt.Println("  ERROR:", err)
		} else {
			fmt.Printf("  OK: id=%d name=%q currency=%s manager=%v test=%v\n",
				c.ID, c.DescriptiveName, c.CurrencyCode, c.Manager, c.TestAccount)
		}
	}
}
