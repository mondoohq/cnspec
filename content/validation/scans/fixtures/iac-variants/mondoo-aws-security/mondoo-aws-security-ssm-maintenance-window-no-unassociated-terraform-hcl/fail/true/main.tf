# Non-compliant: maintenance window allows unassociated targets.
resource "aws_ssm_maintenance_window" "fail_example" {
  name                       = "example-window"
  schedule                   = "cron(0 16 ? * TUE *)"
  duration                   = 3
  cutoff                     = 1
  allow_unassociated_targets = true
}
