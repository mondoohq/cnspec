# Prompts and responses are delivered to CloudWatch Logs.
resource "aws_bedrock_model_invocation_logging_configuration" "example" {
  logging_config {
    text_data_delivery_enabled = true

    cloudwatch_config {
      log_group_name = "bedrock-invocations"
      role_arn       = "arn:aws:iam::111122223333:role/bedrock-logging"
    }
  }
}
