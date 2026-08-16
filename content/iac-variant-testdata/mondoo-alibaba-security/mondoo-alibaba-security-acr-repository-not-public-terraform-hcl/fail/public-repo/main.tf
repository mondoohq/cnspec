# A PUBLIC repository on an instance with an internet endpoint can be pulled by
# anyone who learns the path, with no principal recorded against the pull.
resource "alicloud_cr_ee_repo" "app" {
  instance_id = "cri-example"
  namespace   = "platform"
  name        = "app"
  summary     = "application images"
  repo_type   = "PUBLIC"
}
