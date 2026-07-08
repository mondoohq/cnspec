resource "gitlab_project" "example" {
  name                                 = "example"
  only_allow_merge_if_pipeline_succeeds = false
}
