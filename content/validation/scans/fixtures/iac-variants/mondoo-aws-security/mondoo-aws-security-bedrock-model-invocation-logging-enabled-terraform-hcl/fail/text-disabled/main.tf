# Only image data is logged, so prompts and responses are not recorded.
resource "aws_bedrock_model_invocation_logging_configuration" "example" {
  logging_config {
    text_data_delivery_enabled  = false
    image_data_delivery_enabled = true

    s3_config {
      bucket_name = "bedrock-invocation-logs"
    }
  }
}
