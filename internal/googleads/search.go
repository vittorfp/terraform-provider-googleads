package googleads

import (
	"context"
	"errors"

	"google.golang.org/api/iterator"

	servicespb "github.com/shenzhencenter/google-ads-pb/services"
)

// Search runs a GAQL query against the GoogleAdsService and accumulates all
// rows into a slice. The underlying iterator pages automatically; for large
// result sets prefer SearchStream directly via the client.
func (c *Client) Search(ctx context.Context, customerID, query string) ([]*servicespb.GoogleAdsRow, error) {
	cl, err := c.googleAdsClient(ctx)
	if err != nil {
		return nil, err
	}
	it := cl.Search(ctx, &servicespb.SearchGoogleAdsRequest{
		CustomerId: customerID,
		Query:      query,
	})
	var rows []*servicespb.GoogleAdsRow
	for {
		row, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return rows, nil
		}
		if err != nil {
			return nil, WrapAPIError(err)
		}
		rows = append(rows, row)
	}
}

// SearchOne runs a GAQL query and returns the first row, or ErrNotFound if
// the result set is empty. Useful for primary-key lookups.
func (c *Client) SearchOne(ctx context.Context, customerID, query string) (*servicespb.GoogleAdsRow, error) {
	rows, err := c.Search(ctx, customerID, query)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0], nil
}
