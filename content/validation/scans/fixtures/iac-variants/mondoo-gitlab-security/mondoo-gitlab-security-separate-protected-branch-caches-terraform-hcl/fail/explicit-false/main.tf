resource "gitlab_project" "example" {
  name                = "example"
  ci_separated_caches = false
}
