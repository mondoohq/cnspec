resource "unifi_wlan" "modern" {
  name          = "corp"
  security      = "wpapsk"
  passphrase    = var.wifi_passphrase
  network_id    = unifi_network.lan.id
  user_group_id = unifi_user_group.default.id
  wpa3_support  = true
}
