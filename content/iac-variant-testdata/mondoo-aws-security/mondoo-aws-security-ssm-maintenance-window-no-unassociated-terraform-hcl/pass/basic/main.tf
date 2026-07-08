# Compliant: maintenance window does not register unassociated targets.
resource "aws_ssm_maintenance_window" "pass_example" {
  name                       = "example-window"
  schedule                   = "cron(0 16 ? * TUE *)"
  duration                   = 3
  cutoff                     = 1
  allow_unassociated_targets = false
}
