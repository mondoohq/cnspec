resource "unifi_setting" "mgmt" {
  site = "default"

  mgmt = {
    ssh_enabled = true
  }
}
