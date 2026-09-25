package googleads

import (
	"context"

	pbclients "github.com/shenzhencenter/google-ads-pb/clients"
	"github.com/shenzhencenter/google-ads-pb/services"
)

// ListAccessibleCustomers returns the resource names of every customer
// directly accessible to the OAuth user — i.e., what `customer_client_link`s
// say this user has. No login-customer-id required.
func (c *Client) ListAccessibleCustomers(ctx context.Context) ([]string, error) {
	c.mu.Lock()
	if c.customers == nil {
		cl, err := pbclients.NewCustomerClient(ctx, c.opts...)
		if err != nil {
			c.mu.Unlock()
			return nil, WrapAPIError(err)
		}
		c.customers = cl
	}
	cl := c.customers
	c.mu.Unlock()

	resp, err := cl.ListAccessibleCustomers(ctx, &services.ListAccessibleCustomersRequest{})
	if err != nil {
		return nil, WrapAPIError(err)
	}
	return resp.GetResourceNames(), nil
}
