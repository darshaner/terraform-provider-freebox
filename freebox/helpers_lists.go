package freebox

import "github.com/hashicorp/terraform-plugin-framework/types"

// expandStringList converts []types.String -> []string, preserving empty entries as "".
func expandStringList(in []types.String) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(in))
	for i, s := range in {
		if s.IsNull() || s.IsUnknown() {
			out[i] = ""
		} else {
			out[i] = s.ValueString()
		}
	}
	return out
}

// flattenStringList converts []string -> []types.String, turning "" into types.StringNull().
func flattenStringList(in []string) []types.String {
	if in == nil {
		return nil
	}
	out := make([]types.String, len(in))
	for i, s := range in {
		if s == "" {
			out[i] = types.StringNull()
		} else {
			out[i] = types.StringValue(s)
		}
	}
	return out
}
