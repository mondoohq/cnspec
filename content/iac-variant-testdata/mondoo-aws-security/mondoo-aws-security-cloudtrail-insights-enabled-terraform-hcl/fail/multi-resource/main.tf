# Two trails; the second only enables one insight type, so .all() must fail.
resource "aws_cloudtrail" "complete" {
  name           = "complete"
  s3_bucket_name = "example-bucket"
  insight_selector {
    insight_type = "ApiCallRateInsight"
  }
  insight_selector {
    insight_type = "ApiErrorRateInsight"
  }
}

resource "aws_cloudtrail" "partial" {
  name           = "partial"
  s3_bucket_name = "example-bucket"
  insight_selector {
    insight_type = "ApiCallRateInsight"
  }
}
