# With immutability off, app:1.4.2 can be repointed at different content after it
# was reviewed and deployed, and every later pull of that tag gets the new image.
resource "alicloud_cr_ee_repo" "app" {
  instance_id      = "cri-example"
  namespace        = "platform"
  name             = "app"
  summary          = "application images"
  repo_type        = "PRIVATE"
  tag_immutability = false
}
