package googleads

import "context"

type CustomerView struct {
	ResourceName    string
	ID              int64
	DescriptiveName string
	CurrencyCode    string
	TimeZone        string
	Manager         bool
	TestAccount     bool
}

// GetCustomer reads the named customer's basic metadata.
func (c *Client) GetCustomer(ctx context.Context, customerID string) (*CustomerView, error) {
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			customer.resource_name,
			customer.id,
			customer.descriptive_name,
			customer.currency_code,
			customer.time_zone,
			customer.manager,
			customer.test_account
		FROM customer
		WHERE customer.id = `+customerID)
	if err != nil {
		return nil, err
	}
	u := row.GetCustomer()
	return &CustomerView{
		ResourceName:    u.GetResourceName(),
		ID:              u.GetId(),
		DescriptiveName: u.GetDescriptiveName(),
		CurrencyCode:    u.GetCurrencyCode(),
		TimeZone:        u.GetTimeZone(),
		Manager:         u.GetManager(),
		TestAccount:     u.GetTestAccount(),
	}, nil
}
