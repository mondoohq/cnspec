# Non-compliant: two clusters; exactly one omits the Fargate KMS key. .all()
# over resources must still fail.
resource "aws_ecs_cluster" "good" {
  name = "good"

  configuration {
    managed_storage_configuration {
      fargate_ephemeral_storage_kms_key_id = "arn:aws:kms:us-east-1:111122223333:key/abcd"
    }
  }
}

resource "aws_ecs_cluster" "bad" {
  name = "bad"

  configuration {
    managed_storage_configuration {
    }
  }
}
