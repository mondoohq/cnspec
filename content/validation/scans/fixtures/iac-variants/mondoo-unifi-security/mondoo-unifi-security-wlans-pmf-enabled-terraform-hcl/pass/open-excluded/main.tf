resource "unifi_wlan" "guest" {
  name          = "guest"
  security      = "open"
  network_id    = unifi_network.guest.id
  user_group_id = unifi_user_group.default.id
}
