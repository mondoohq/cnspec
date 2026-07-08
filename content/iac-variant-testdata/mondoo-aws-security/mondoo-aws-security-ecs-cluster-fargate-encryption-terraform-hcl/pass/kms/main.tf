# Compliant: Fargate ephemeral storage encrypted with a KMS key.
resource "aws_ecs_cluster" "pass_example" {
  name = "pass-example"

  configuration {
    managed_storage_configuration {
      fargate_ephemeral_storage_kms_key_id = "arn:aws:kms:us-east-1:111122223333:key/abcd"
    }
  }
}
