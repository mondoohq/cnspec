resource "unifi_wlan" "guest" {
  name          = "guest"
  security      = "wpapsk"
  passphrase    = var.guest_passphrase
  is_guest      = true
  l2_isolation  = true
  network_id    = unifi_network.guest.id
  user_group_id = data.unifi_client_qos_rate.default.id
}
