resource "gitlab_project" "example" {
  name                                        = "example"
  ci_pipeline_variables_minimum_override_role = "developer"
}
