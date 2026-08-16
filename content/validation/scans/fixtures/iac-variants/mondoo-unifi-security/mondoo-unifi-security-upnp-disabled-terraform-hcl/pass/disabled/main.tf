resource "unifi_setting" "gateway" {
  site = "default"

  usg = {
    upnp_enabled = false
    syn_cookies  = true
  }
}
