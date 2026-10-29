resource "unifi_setting" "gateway" {
  site = "default"

  usg = {
    syn_cookies = true

    upnp = {
      enabled = false
    }
  }
}
