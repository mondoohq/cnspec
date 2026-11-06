resource "stackit_modelexperiments_instance" "tracking" {
  project_id                   = "00000000-0000-0000-0000-000000000000"
  name                         = "ml-tracking"
  deleted_experiment_retention = "1d"
}
