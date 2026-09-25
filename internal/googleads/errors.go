package googleads

import (
	"errors"
	"fmt"
	"strings"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	errorspb "github.com/shenzhencenter/google-ads-pb/errors"
)

// APIError wraps a gRPC error from the Google Ads API and exposes the
// per-field GoogleAdsFailure details so resource diagnostics can highlight
// which Terraform attribute the API rejected.
type APIError struct {
	Code    string             // top-level gRPC code, e.g. INVALID_ARGUMENT
	Message string             // top-level message
	Details []APIErrorDetail   // one entry per GoogleAdsError in the failure
	Errors  []string           // backwards-compat: formatted "field: reason" lines (use Details for structured access)
}

// APIErrorDetail is the structured form of a single GoogleAdsError. The
// per-field decomposition lets resources emit one Terraform diagnostic
// per actual issue (so a user with three bad fields gets three pointed
// errors instead of one wall of text).
type APIErrorDetail struct {
	// FieldPath is the dotted path the API reports, with `[N]` for list
	// indexes (e.g. `operations[0].create.headlines[3].text`). Useful as
	// the Detail line on the diagnostic.
	FieldPath string
	// LeafField is the last segment of FieldPath, intended for matching
	// against Terraform attribute names. For most resources this is the
	// HCL attribute name verbatim; some need a small remap (e.g. the API
	// says `target_cpa.target_cpa_micros` but the HCL attribute is just
	// `target_cpa_micros` — the resource code can use LeafField directly
	// or apply its own mapping).
	LeafField string
	// Message is the human-readable explanation from the API.
	Message string
	// ErrorCode is the lowercase oneof name from GoogleAdsError.ErrorCode
	// (e.g. "field_error", "string_length_error"). Useful for telemetry
	// or branching on specific error classes.
	ErrorCode string
}

func (e *APIError) Error() string {
	if len(e.Errors) == 0 {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s\n  - %s", e.Code, e.Message, strings.Join(e.Errors, "\n  - "))
}

// WrapAPIError converts a raw gRPC error into an *APIError with structured
// details. Returns nil if err is nil. Returns the original err unwrapped if
// it is not a gRPC status error.
func WrapAPIError(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	out := &APIError{Code: st.Code().String(), Message: st.Message()}
	unmarshal := proto.UnmarshalOptions{DiscardUnknown: true}
	for _, detail := range st.Proto().GetDetails() {
		var failure errorspb.GoogleAdsFailure
		if err := unmarshal.Unmarshal(detail.GetValue(), &failure); err != nil {
			continue
		}
		for _, e := range failure.GetErrors() {
			d := newAPIErrorDetail(e)
			out.Details = append(out.Details, d)
			out.Errors = append(out.Errors, d.formatted())
		}
	}
	return out
}

func newAPIErrorDetail(e *errorspb.GoogleAdsError) APIErrorDetail {
	d := APIErrorDetail{Message: e.GetMessage()}
	if loc := e.GetLocation(); loc != nil {
		var segments []string
		for _, fp := range loc.GetFieldPathElements() {
			seg := fp.GetFieldName()
			if fp.Index != nil {
				seg = fmt.Sprintf("%s[%d]", seg, fp.GetIndex())
			}
			segments = append(segments, seg)
		}
		if len(segments) > 0 {
			d.FieldPath = strings.Join(segments, ".")
			// LeafField = last segment with any [N] stripped — useful
			// because Terraform attributes don't usually carry list
			// indexes in their names.
			last := segments[len(segments)-1]
			if i := strings.IndexByte(last, '['); i >= 0 {
				last = last[:i]
			}
			d.LeafField = last
		}
	}
	if code := e.GetErrorCode(); code != nil {
		// ErrorCode is a oneof — find which variant is populated.
		// Reflecting on the descriptor is overkill; for v1 we ship
		// just the message and let the user grep if needed.
		d.ErrorCode = "google_ads_error"
	}
	return d
}

// formatted is the backwards-compat single-line representation that
// APIError.Errors used to carry. Same format: "field.path: message".
func (d APIErrorDetail) formatted() string {
	if d.FieldPath == "" {
		return d.Message
	}
	return d.FieldPath + ": " + d.Message
}

// IsNotFound reports whether err is or wraps ErrNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
