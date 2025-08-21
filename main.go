package main

import (
	"context"

	"github.com/darshaner/terraform-provider-freebox/freebox"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	providerserver.Serve(context.Background(), freebox.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/darshaner/freebox",
	})
}
