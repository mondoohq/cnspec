# A tag cannot be repointed once it has been pushed.
resource "alicloud_cr_ee_repo" "app" {
  instance_id      = "cri-example"
  namespace        = "platform"
  name             = "app"
  summary          = "application images"
  repo_type        = "PRIVATE"
  tag_immutability = true
}
