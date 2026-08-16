resource "unifi_setting" "gateway" {
  site = "default"

  usg = {
    syn_cookies  = false
    upnp_enabled = false
  }
}
