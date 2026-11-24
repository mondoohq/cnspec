resource "gitlab_project" "example" {
  name                                      = "example"
  allow_pipeline_trigger_approve_deployment = false
}
