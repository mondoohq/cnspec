resource "unifi_setting" "gateway" {
  site = "default"

  usg = {
    upnp = {
      enabled = true
    }
  }
}
