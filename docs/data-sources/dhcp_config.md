# freebox_dhcp_config (Data Source)

Fetches the current DHCP server configuration.

## Example Usage

```hcl
data "freebox_dhcp_config" "current" {}

output "dhcp_config" {
  value = data.freebox_dhcp_config.current
}
````

## Attribute Reference

* **enabled** (Bool)
* **sticky\_assign** (Bool)
* **gateway** (String)
* **netmask** (String)
* **ip\_range\_start** (String)
* **ip\_range\_end** (String)
* **always\_broadcast** (Bool)
* **ignore\_out\_of\_range\_hint** (Bool)
* **dns** (List of String)