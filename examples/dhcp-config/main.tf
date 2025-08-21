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

resource "freebox_dhcp_config" "lan" {
  enabled                  = true
  sticky_assign            = true
  ip_range_start           = "192.168.0.2"
  ip_range_end             = "192.168.0.50"
  always_broadcast         = false
  ignore_out_of_range_hint = false
  dns                      = ["192.168.0.254"]
}

output "gateway" {
  value = freebox_dhcp_config.lan.gateway
}
