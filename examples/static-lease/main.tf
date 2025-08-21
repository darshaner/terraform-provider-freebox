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

resource "freebox_dhcp_lease" "workstation" {
  mac     = "AA:BB:CC:DD:EE:FF"
  ip      = "192.168.0.42"
  comment = "Dev workstation"
}

output "lease_hostname" {
  value = freebox_dhcp_lease.workstation.hostname
}
