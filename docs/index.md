# Freebox Provider

The Freebox provider allows you to manage DHCP static leases and DHCP server configuration.

## Example Usage

```hcl
provider "freebox" {
  app_token = var.freebox_app_token
  # base_url = "http://mafreebox.freebox.fr/api/v8" # optional
}
````

## Schema

### Required

* **app\_token** (String) Freebox application token (after approving the app on the Freebox).

### Optional

* **base\_url** (String) Freebox API base URL. Defaults to `http://mafreebox.freebox.fr/api/v8`.