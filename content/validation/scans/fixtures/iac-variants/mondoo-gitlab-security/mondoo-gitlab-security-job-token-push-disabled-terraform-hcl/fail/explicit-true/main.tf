resource "gitlab_project" "example" {
  name                                     = "example"
  ci_push_repository_for_job_token_allowed = true
}
