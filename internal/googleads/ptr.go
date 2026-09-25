package googleads

// Small helpers for the *string / *int64 / *bool pointer fields the Google
// Ads protos use for proto3-optional / oneof scalar wrappers.

func StringPtr(s string) *string { return &s }
func Int64Ptr(i int64) *int64    { return &i }
func BoolPtr(b bool) *bool       { return &b }

func StringOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func Int64OrZero(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func BoolOrFalse(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}
