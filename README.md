# Freebox Terraform Provider

Manage Freebox DHCP **static leases** and **DHCP server configuration** with Terraform.

## Installation

Build the provider locally:

```bash
go mod tidy
go build -o ~/.terraform.d/plugins/registry.terraform.io/local/freebox/1.0.0/$(uname -s | tr '[:upper:]' '[:lower:]')_$(uname -m)/terraform-provider-freebox
````

Use it in Terraform:

```hcl
terraform {
  required_providers {
    freebox = {
      source  = "registry.terraform.io/darshaner/freebox"
      version = "1.0.0"
    }
  }
}

provider "freebox" {
  app_token = var.freebox_app_token
  # base_url = "http://mafreebox.freebox.fr/api/v15" # override if needed
}
```

## Getting an `app_token` (one‑time)

Use the helper script:

```bash
python3 tools/freebox_api_token.py
```

Approve on your Freebox screen when prompted, then copy the printed **APP TOKEN**.

> Keep the token secret (store in a secret manager or environment variable).

### Configuring permissions

At the time of this writing, the managment of permissions can not be done via the API. It must be done manually through the freebox OS web UI.

If you need to change the default set of permissions, first head to [http://mafreebox.freebox.fr](http://mafreebox.freebox.fr) and log in.

Then open the `Paramètres de la Freebox` menu, double click on `Gestion des accès` and switch to the `Applications` tab.

You should see the application you just registered earlier ; click on the `Editer` icon `🖉`.

Finally, pick the permissions your application requires. For a basic usage the following ones are good enough:

- `Accès au gestionnaire de téléchargements`
- `Accès aux fichiers de la Freebox`
- `Modification des réglages de la Freebox`
- `Contrôle de la VM`

## Resources

### `freebox_dhcp_lease`

```hcl
resource "freebox_dhcp_lease" "example" {
  mac     = "AA:BB:CC:DD:EE:FF"  # id == mac
  ip      = "192.168.1.42"
  comment = "example device"
}
```

### `freebox_dhcp_config` (singleton)

```hcl
resource "freebox_dhcp_config" "main" {
  enabled                  = true
  sticky_assign            = true
  ip_range_start           = "192.168.1.2"
  ip_range_end             = "192.168.1.50"
  always_broadcast         = false
  ignore_out_of_range_hint = false
  dns                      = ["192.168.1.254", "", "", "", ""]
}
```

## Data Sources

```hcl
data "freebox_dhcp_config" "current" {}

data "freebox_dhcp_leases" "all" {}
```

## Notes

* `gateway` and `netmask` are **read‑only** on DHCP config.
* Lease `id` equals `mac`.
* API error codes (e.g., `inval_ip_range`, `inval_gw_net`) are surfaced with human‑friendly messages.

## License

MIT

