# freebox_dhcp_leases (Data Source)

Fetches the list of DHCP static leases.

## Example Usage

```hcl
data "freebox_dhcp_leases" "all" {}

output "leases" {
  value = data.freebox_dhcp_leases.all.leases
}
````

## Attribute Reference

* **leases** (List of Object)

  * **id** (String)
  * **mac** (String)
  * **ip** (String)
  * **comment** (String)
  * **hostname** (String)
  * **host** (Object)
