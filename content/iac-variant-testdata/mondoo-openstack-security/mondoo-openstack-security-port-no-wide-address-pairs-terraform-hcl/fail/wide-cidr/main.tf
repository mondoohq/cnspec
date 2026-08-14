# A /0 allowed address pair lets the port spoof any source address, defeating
# anti-spoofing enforcement entirely.
resource "openstack_networking_port_v2" "vip" {
  name           = "keepalived-vip"
  network_id     = openstack_networking_network_v2.internal.id
  admin_state_up = true

  allowed_address_pairs {
    ip_address = "0.0.0.0/0"
  }
}
