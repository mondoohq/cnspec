resource "unifi_wlan" "guest" {
  name          = "guest"
  security      = "wpapsk"
  passphrase    = var.guest_passphrase
  is_guest      = false
  network_id    = unifi_network.lan.id
  user_group_id = data.unifi_client_qos_rate.default.id
}
