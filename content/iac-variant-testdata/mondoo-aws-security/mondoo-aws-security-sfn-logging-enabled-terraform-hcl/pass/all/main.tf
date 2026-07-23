# Compliant: state machine logging is enabled at the ALL level.
resource "aws_sfn_state_machine" "example" {
  name     = "example-state-machine"
  role_arn = "arn:aws:iam::111122223333:role/service-role/StepFunctions"

  definition = jsonencode({
    Comment = "example"
    StartAt = "Done"
    States  = { Done = { Type = "Succeed" } }
  })

  logging_configuration {
    log_destination        = "arn:aws:logs:us-east-1:111122223333:log-group:/aws/sfn/example:*"
    include_execution_data = true
    level                  = "ALL"
  }
}
