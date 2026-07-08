resource "unifi_setting" "gateway" {
  site = "default"

  usg = {
    receive_redirects = true
  }
}
