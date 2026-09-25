package googleads

import "context"

// CustomerClient mirrors the practical subset of customer_client rows
// surfaced by the Ads API when walking an MCC's tree.
type CustomerClient struct {
	ID              int64
	DescriptiveName string
	Manager         bool
	TestAccount     bool
	Level           int64
	Hidden          bool
}

// ListLeafCustomerClients returns every non-removed, non-manager,
// non-hidden child account under the given MCC. Useful for fanning
// `googleads-tfgen` out across a whole MCC in one command.
//
// `mccCustomerID` is both the customer_id (whose tree we walk) and
// implicitly the login-customer-id — the caller is expected to have
// built the Client with LoginCustomerID set to the same value.
func (c *Client) ListLeafCustomerClients(ctx context.Context, mccCustomerID string) ([]CustomerClient, error) {
	rows, err := c.Search(ctx, mccCustomerID, `
		SELECT
			customer_client.id,
			customer_client.descriptive_name,
			customer_client.manager,
			customer_client.test_account,
			customer_client.level,
			customer_client.hidden,
			customer_client.status
		FROM customer_client
		WHERE customer_client.manager = false
		  AND customer_client.status = 'ENABLED'`)
	if err != nil {
		return nil, err
	}
	out := make([]CustomerClient, 0, len(rows))
	for _, row := range rows {
		cc := row.GetCustomerClient()
		if cc.GetHidden() {
			continue
		}
		out = append(out, CustomerClient{
			ID:              cc.GetId(),
			DescriptiveName: cc.GetDescriptiveName(),
			Manager:         cc.GetManager(),
			TestAccount:     cc.GetTestAccount(),
			Level:           cc.GetLevel(),
			Hidden:          cc.GetHidden(),
		})
	}
	return out, nil
}
