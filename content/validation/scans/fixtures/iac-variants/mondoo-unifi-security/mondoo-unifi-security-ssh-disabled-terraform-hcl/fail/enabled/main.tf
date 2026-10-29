resource "unifi_setting" "mgmt" {
  site = "default"

  mgmt = {
    ssh = {
      enabled = true
    }
  }
}
