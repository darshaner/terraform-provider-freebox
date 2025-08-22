# freebox\_port\_forward

Manages a **port forwarding rule** on the Freebox.

## Example Usage

```hcl
resource "freebox_port_forward" "ssh" {
  enabled        = true
  ip_proto       = "tcp"
  wan_port_start = 2222
  wan_port_end   = 2222
  lan_ip         = "192.168.0.100"
  lan_port       = 22
  src_ip         = "0.0.0.0"
  comment        = "SSH forwarding"
}
```

## Argument Reference

* **enabled** (Bool, Optional, Default: `true`) Enable/disable this forwarding rule.
* **ip\_proto** (String, Optional, Default: `"tcp"`) IP protocol. One of: `tcp`, `udp`.
* **wan\_port\_start** (Number, Required) External (WAN) start port.
* **wan\_port\_end** (Number, Required) External (WAN) end port.
* **lan\_ip** (String, Required) Target **LAN IP** for the forwarding.
* **lan\_port** (Number, Required) Target **LAN start port**. The last port is `lan_port + wan_port_end - wan_port_start`.
* **src\_ip** (String, Optional, Default: `"0.0.0.0"`) Source IP filter. Use `0.0.0.0` to accept any source.
* **comment** (String, Optional) Free-form comment/label for the rule.

## Attribute Reference

* **id** (Number) Rule identifier assigned by the Freebox.
* **hostname** (String) Resolved hostname of the LAN target (read-only).

## Import

Import by rule **id**:

```bash
terraform import freebox_port_forward.ssh 7
```
