# Compliant: state machine uses a customer-managed KMS key for encryption.
resource "aws_sfn_state_machine" "example" {
  name     = "example-state-machine"
  role_arn = "arn:aws:iam::111122223333:role/service-role/StepFunctions"

  definition = jsonencode({
    Comment = "example"
    StartAt = "Done"
    States  = { Done = { Type = "Succeed" } }
  })

  encryption_configuration {
    type                              = "CUSTOMER_MANAGED_KMS_KEY"
    kms_key_id                        = "arn:aws:kms:us-east-1:111122223333:key/abcd-1234"
    kms_data_key_reuse_period_seconds = 300
  }
}
