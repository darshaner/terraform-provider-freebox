# freebox\_port\_forwardings

Reads all **port forwarding rules** from the Freebox.

## Example Usage

```hcl
data "freebox_port_forwardings" "all" {}

output "all_forwards" {
  value = data.freebox_port_forwardings.all.forwards
}
```

## Attribute Reference

* **id** (String) Synthetic identifier for this data source.
* **forwards** (List of Objects) All forwarding rules returned by the Freebox.
  Each object contains:

  * **id** (Number) Rule identifier.
  * **enabled** (Bool) Whether the rule is enabled.
  * **ip\_proto** (String) IP protocol (`tcp` or `udp`).
  * **wan\_port\_start** (Number) External (WAN) start port.
  * **wan\_port\_end** (Number) External (WAN) end port.
  * **lan\_ip** (String) Target LAN IP.
  * **lan\_port** (Number) Target LAN start port.
  * **src\_ip** (String) Source IP filter (`0.0.0.0` for any).
  * **comment** (String) Rule comment, if any.
  * **hostname** (String) Resolved hostname of the LAN target (read-only).