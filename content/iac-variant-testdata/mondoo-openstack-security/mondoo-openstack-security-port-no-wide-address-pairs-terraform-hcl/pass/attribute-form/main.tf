# Same setting written as an attribute list rather than repeated blocks.
resource "openstack_networking_port_v2" "vip" {
  name           = "keepalived-vip"
  network_id     = openstack_networking_network_v2.internal.id
  admin_state_up = true

  allowed_address_pairs = [
    {
      ip_address = "10.0.10.42"
    },
    {
      ip_address = "10.0.10.43"
    },
  ]
}
