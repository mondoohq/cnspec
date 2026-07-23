# Compliant: both insight types are enabled through a dynamic block.
variable "insight_types" {
  type    = list(string)
  default = ["ApiCallRateInsight", "ApiErrorRateInsight"]
}

resource "aws_cloudtrail" "example" {
  name           = "example"
  s3_bucket_name = "example-bucket"

  dynamic "insight_selector" {
    for_each = var.insight_types
    content {
      insight_type = insight_selector.value
    }
  }
}
