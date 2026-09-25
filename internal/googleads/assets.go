package googleads

import (
	"context"
	"fmt"
	"net/http"

	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pbclients "github.com/shenzhencenter/google-ads-pb/clients"
	"github.com/shenzhencenter/google-ads-pb/common"
	"github.com/shenzhencenter/google-ads-pb/enums"
	"github.com/shenzhencenter/google-ads-pb/resources"
	"github.com/shenzhencenter/google-ads-pb/services"
)

// AssetInput is the v1 surface for assets. v1 supports two payload types —
// text and image. Other payload variants (YouTube, sitelinks, callouts, etc.)
// are out of scope until the corresponding resource ships.
type AssetInput struct {
	CustomerID string
	Name       string // optional; the API will auto-name if empty

	// Exactly one of these must be set per input.
	Text      string // for TEXT assets
	ImageData []byte // for IMAGE assets
}

// AssetView is the read-back shape.
type AssetView struct {
	ResourceName string
	ID           int64
	Name         string
	Type         string // TEXT | IMAGE | ...
	Text         string // populated for TEXT assets
	// Image bytes are never returned by the API (the `data` field is
	// mutate-only), so we surface only the dimensions / file size when
	// present and let drift checks compare metadata, not content.
	ImageFileSize int64
	ImageMimeType string
}

// CreateTextAsset uploads a text asset and returns the resource name.
func (c *Client) CreateTextAsset(ctx context.Context, in AssetInput) (string, error) {
	asset := &resources.Asset{
		AssetData: &resources.Asset_TextAsset{
			TextAsset: &common.TextAsset{Text: StringPtr(in.Text)},
		},
	}
	if in.Name != "" {
		asset.Name = StringPtr(in.Name)
	}
	return c.mutateAssetCreate(ctx, in.CustomerID, asset)
}

// CreateImageAsset uploads an image asset. The MIME type is inferred from
// the byte content via http.DetectContentType — callers don't need to set
// it explicitly.
func (c *Client) CreateImageAsset(ctx context.Context, in AssetInput) (string, error) {
	mime, err := detectImageMime(in.ImageData)
	if err != nil {
		return "", err
	}
	asset := &resources.Asset{
		AssetData: &resources.Asset_ImageAsset{
			ImageAsset: &common.ImageAsset{
				Data:     in.ImageData,
				MimeType: mime,
			},
		},
	}
	if in.Name != "" {
		asset.Name = StringPtr(in.Name)
	}
	return c.mutateAssetCreate(ctx, in.CustomerID, asset)
}

// GetAsset reads the named asset. Returns ErrNotFound if it doesn't exist
// (or was removed from the account).
func (c *Client) GetAsset(ctx context.Context, resourceName string) (*AssetView, error) {
	customerID, _, err := ParseResourceName(resourceName, "assets")
	if err != nil {
		return nil, err
	}
	row, err := c.SearchOne(ctx, customerID, `
		SELECT
			asset.resource_name,
			asset.id,
			asset.name,
			asset.type,
			asset.text_asset.text,
			asset.image_asset.file_size,
			asset.image_asset.mime_type
		FROM asset
		WHERE asset.resource_name = '`+resourceName+`'`)
	if err != nil {
		return nil, err
	}
	a := row.GetAsset()
	return &AssetView{
		ResourceName:  a.GetResourceName(),
		ID:            a.GetId(),
		Name:          a.GetName(),
		Type:          a.GetType().String(),
		Text:          a.GetTextAsset().GetText(),
		ImageFileSize: a.GetImageAsset().GetFileSize(),
		ImageMimeType: a.GetImageAsset().GetMimeType().String(),
	}, nil
}

// UpdateAsset is intentionally narrow — the Ads API treats most asset
// fields as immutable once created. v1 supports renaming only.
func (c *Client) UpdateAsset(ctx context.Context, resourceName string, name string) error {
	cl, err := c.assetClient(ctx)
	if err != nil {
		return err
	}
	customerID, _, err := ParseResourceName(resourceName, "assets")
	if err != nil {
		return err
	}
	asset := &resources.Asset{ResourceName: resourceName, Name: StringPtr(name)}
	_, err = cl.MutateAssets(ctx, &services.MutateAssetsRequest{
		CustomerId: customerID,
		Operations: []*services.AssetOperation{{
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
			Operation:  &services.AssetOperation_Update{Update: asset},
		}},
	})
	return WrapAPIError(err)
}

func (c *Client) mutateAssetCreate(ctx context.Context, customerID string, asset *resources.Asset) (string, error) {
	cl, err := c.assetClient(ctx)
	if err != nil {
		return "", err
	}
	resp, err := cl.MutateAssets(ctx, &services.MutateAssetsRequest{
		CustomerId: customerID,
		Operations: []*services.AssetOperation{{
			Operation: &services.AssetOperation_Create{Create: asset},
		}},
	})
	if err != nil {
		return "", WrapAPIError(err)
	}
	return resp.GetResults()[0].GetResourceName(), nil
}

func (c *Client) assetClient(ctx context.Context) (*pbclients.AssetClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.assets != nil {
		return c.assets, nil
	}
	cl, err := pbclients.NewAssetClient(ctx, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("googleads: asset client: %w", err)
	}
	c.assets = cl
	return cl, nil
}

// detectImageMime maps the file's content to the AdsAPI MIME enum. Returns
// an error if the bytes don't sniff as a supported image format.
func detectImageMime(data []byte) (enums.MimeTypeEnum_MimeType, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("googleads: image data is empty")
	}
	sniff := data
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	switch http.DetectContentType(sniff) {
	case "image/png":
		return enums.MimeTypeEnum_IMAGE_PNG, nil
	case "image/jpeg":
		return enums.MimeTypeEnum_IMAGE_JPEG, nil
	case "image/gif":
		return enums.MimeTypeEnum_IMAGE_GIF, nil
	default:
		return 0, fmt.Errorf("googleads: unsupported image format — supported: PNG, JPEG, GIF")
	}
}
