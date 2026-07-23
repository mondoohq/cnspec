resource "unifi_setting" "mgmt" {
  site = "default"

  mgmt = {
    auto_upgrade = true
  }
}
