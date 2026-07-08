# Non-compliant: trail enables only the API error rate insight, missing the
# API call rate insight.
resource "aws_cloudtrail" "fail_example" {
  name           = "example"
  s3_bucket_name = "example-bucket"

  insight_selector {
    insight_type = "ApiErrorRateInsight"
  }
}
