resource "gitlab_project" "example" {
  name                                             = "example"
  only_allow_merge_if_all_discussions_are_resolved = false
}
