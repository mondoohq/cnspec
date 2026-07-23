resource "unifi_setting" "gateway" {
  site = "default"

  usg = {
    syn_cookies    = true
    broadcast_ping = false
  }
}
