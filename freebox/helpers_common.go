package freebox

import "github.com/hashicorp/terraform-plugin-framework/types"

// stringOrNull turns an empty string into a Terraform null string
func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
