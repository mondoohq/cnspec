resource "gitlab_project" "example" {
  name                            = "example"
  allow_merge_on_skipped_pipeline = false
}
