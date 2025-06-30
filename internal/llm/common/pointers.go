package common

// pointer helper functions for optional JSON fields; commonly needed for opt parameters

// BoolPtr returns a pointer to a bool value
func BoolPtr(b bool) *bool {
	return &b
}

// IntPtr returns a pointer to an int value
func IntPtr(i int) *int {
	return &i
}

// Float64Ptr returns a pointer to a float64 value
func Float64Ptr(f float64) *float64 {
	return &f
}

// StringPtr returns a pointer to a string value
func StringPtr(s string) *string {
	return &s
}
