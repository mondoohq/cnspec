resource "unifi_wlan" "internal" {
  name          = "internal"
  security      = "wpapsk"
  passphrase    = var.wifi_passphrase
  network_id    = unifi_network.lan.id
  user_group_id = data.unifi_client_qos_rate.default.id
}
