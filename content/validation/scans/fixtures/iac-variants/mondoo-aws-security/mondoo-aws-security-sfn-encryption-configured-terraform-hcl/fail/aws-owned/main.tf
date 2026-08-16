# Non-compliant: encryption_configuration explicitly uses the AWS-owned key.
resource "aws_sfn_state_machine" "example" {
  name     = "example-state-machine"
  role_arn = "arn:aws:iam::111122223333:role/service-role/StepFunctions"

  definition = jsonencode({
    Comment = "example"
    StartAt = "Done"
    States  = { Done = { Type = "Succeed" } }
  })

  encryption_configuration {
    type = "AWS_OWNED_KEY"
  }
}
