resource "unifi_wlan" "enterprise" {
  name          = "enterprise"
  security      = "wpaeap"
  network_id    = unifi_network.lan.id
  user_group_id = data.unifi_client_qos_rate.default.id
}
