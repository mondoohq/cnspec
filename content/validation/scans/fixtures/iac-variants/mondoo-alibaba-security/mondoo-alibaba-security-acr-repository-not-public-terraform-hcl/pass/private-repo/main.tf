# Pulling this repository requires credentials.
resource "alicloud_cr_ee_repo" "app" {
  instance_id = "cri-example"
  namespace   = "platform"
  name        = "app"
  summary     = "application images"
  repo_type   = "PRIVATE"
}
