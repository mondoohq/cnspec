# Non-compliant: trail only enables API call rate insight, missing error rate.
resource "aws_cloudtrail" "fail_example" {
  name           = "example"
  s3_bucket_name = "example-bucket"

  insight_selector {
    insight_type = "ApiCallRateInsight"
  }
}
