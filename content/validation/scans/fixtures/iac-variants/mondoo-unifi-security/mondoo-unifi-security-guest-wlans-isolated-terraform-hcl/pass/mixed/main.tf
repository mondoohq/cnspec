resource "unifi_wlan" "guest" {
  name          = "Guest-WiFi"
  security      = "wpapsk"
  passphrase    = var.guest_passphrase
  is_guest      = true
  network_id    = unifi_network.guest.id
  user_group_id = data.unifi_client_qos_rate.default.id
}

resource "unifi_wlan" "corp" {
  name          = "corp"
  security      = "wpaeap"
  network_id    = unifi_network.lan.id
  user_group_id = data.unifi_client_qos_rate.default.id
}
