resource "unifi_wlan" "public" {
  name          = "public"
  security      = "open"
  network_id    = unifi_network.guest.id
  user_group_id = data.unifi_client_qos_rate.default.id
}
