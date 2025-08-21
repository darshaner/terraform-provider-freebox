terraform {
  required_providers {
    freebox = {
      source  = "registry.terraform.io/local/freebox"
      version = "1.0.0"
    }
  }
}

provider "freebox" {
  app_token = var.freebox_app_token
}

data "freebox_dhcp_config" "current" {}

data "freebox_dhcp_leases" "all" {}

output "dhcp_enabled" {
  value = data.freebox_dhcp_config.current.enabled
}

output "leases" {
  value = data.freebox_dhcp_leases.all.leases
}
