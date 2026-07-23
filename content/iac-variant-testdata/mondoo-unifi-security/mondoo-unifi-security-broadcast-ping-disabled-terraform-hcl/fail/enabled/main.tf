resource "unifi_setting" "gateway" {
  site = "default"

  usg = {
    broadcast_ping = true
  }
}
