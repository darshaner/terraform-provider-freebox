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
