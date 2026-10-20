resource "aws_lambda_function" "rotation" {
  function_name = "secret-rotation"
  role          = "arn:aws:iam::123456789012:role/lambda"
  handler       = "index.handler"
  runtime       = "python3.12"
  filename      = "rotation.zip"
}

resource "aws_secretsmanager_secret" "db" {
  name = "db-credentials"
}

resource "aws_secretsmanager_secret_rotation" "db" {
  secret_id           = aws_secretsmanager_secret.db.id
  rotation_lambda_arn = aws_lambda_function.rotation.arn

  rotation_rules {
    automatically_after_days = 30
  }
}
