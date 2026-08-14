# The Alibaba Cloud default of 8 is well inside offline cracking range for a
# modern GPU rig.
resource "alicloud_ram_account_password_policy" "default" {
  minimum_password_length = 8
}
