package provider

import "github.com/hashicorp/terraform-plugin-framework/path"

// attrPath maps a Config field name (snake_case) to a path.Path so that
// validation errors can point at the offending attribute in the provider block.
func attrPath(name string) path.Path {
	return path.Root(name)
}
