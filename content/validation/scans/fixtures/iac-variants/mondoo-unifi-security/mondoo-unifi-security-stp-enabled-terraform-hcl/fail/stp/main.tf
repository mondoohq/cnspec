resource "unifi_setting" "switching" {
  site = "default"

  global_switch = {
    stp_version = "stp"
  }
}
