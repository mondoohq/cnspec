resource "unifi_setting" "gateway" {
  site = "default"

  usg = {
    syn_cookies    = true
    upnp_enabled   = false
    broadcast_ping = false
  }
}
