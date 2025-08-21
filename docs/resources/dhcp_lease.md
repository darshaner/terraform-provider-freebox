# freebox_dhcp_lease (Resource)

Manages a DHCP static lease.

## Example Usage

```hcl
resource "freebox_dhcp_lease" "example" {
  mac     = "AA:BB:CC:DD:EE:FF"
  ip      = "192.168.0.42"
  comment = "My workstation"
}
````

## Argument Reference

* **mac** (String, Required) Host MAC address.
* **ip** (String, Required) IPv4 address to assign to the host.
* **comment** (String, Optional) Optional comment.

## Attribute Reference

* **id** (String) Lease identifier (the MAC address).
* **hostname** (String) Hostname resolved by the Freebox.
* **host** (Object) LAN host information (opaque JSON).

## Import

```shell
terraform import freebox_dhcp_lease.example AA:BB:CC:DD:EE:FF
```