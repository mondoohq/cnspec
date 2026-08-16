resource "aws_secretsmanager_secret" "my_secret" {
  name = "example"
}

resource "aws_secretsmanager_secret_rotation" "this" {
  secret_id           = aws_secretsmanager_secret.my_secret.id
  rotation_lambda_arn = "arn:aws:lambda:us-east-1:123456789012:function:rotate"

  rotation_rules {
    automatically_after_days = 30
  }
}
