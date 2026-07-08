# Compliant: access logging is enabled through a dynamic block that is present
# whenever a logging bucket is configured.
variable "log_bucket" {
  type    = string
  default = "logs.s3.amazonaws.com"
}

resource "aws_cloudfront_distribution" "example" {
  enabled = true

  dynamic "logging_config" {
    for_each = var.log_bucket == "" ? [] : [var.log_bucket]
    content {
      bucket = logging_config.value
      prefix = "cf/"
    }
  }
}
