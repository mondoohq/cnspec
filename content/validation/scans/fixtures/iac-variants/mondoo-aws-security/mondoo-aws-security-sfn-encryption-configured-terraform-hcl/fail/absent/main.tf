# Non-compliant: no encryption_configuration block, so encryption defaults to AWS-owned key.
resource "aws_sfn_state_machine" "example" {
  name     = "example-state-machine"
  role_arn = "arn:aws:iam::111122223333:role/service-role/StepFunctions"

  definition = jsonencode({
    Comment = "example"
    StartAt = "Done"
    States  = { Done = { Type = "Succeed" } }
  })
}
