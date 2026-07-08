resource "unifi_setting" "gateway" {
  site = "default"

  usg = {
    upnp_enabled   = false
    broadcast_ping = false
  }
}
