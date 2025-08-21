# freebox_dhcp_config (Resource)

Manages global DHCP server configuration.

## Example Usage

```hcl
resource "freebox_dhcp_config" "this" {
  enabled                  = true
  sticky_assign            = true
  ip_range_start           = "192.168.0.2"
  ip_range_end             = "192.168.0.50"
  always_broadcast         = false
  ignore_out_of_range_hint = false
  dns                      = ["192.168.0.254"]
}
````

## Argument Reference

* **enabled** (Bool, Required) Enable or disable DHCP server.
* **sticky\_assign** (Bool, Required) Always assign the same IP to a host.
* **ip\_range\_start** (String, Required) Start of DHCP range.
* **ip\_range\_end** (String, Required) End of DHCP range.
* **always\_broadcast** (Bool, Optional) Always broadcast DHCP responses.
* **ignore\_out\_of\_range\_hint** (Bool, Optional) Ignore client-requested IP outside the range.
* **dns** (List of String, Optional) DNS servers to provide in DHCP replies.

## Attribute Reference

* **gateway** (String) Freebox LAN gateway IP.
* **netmask** (String) Netmask of LAN.
