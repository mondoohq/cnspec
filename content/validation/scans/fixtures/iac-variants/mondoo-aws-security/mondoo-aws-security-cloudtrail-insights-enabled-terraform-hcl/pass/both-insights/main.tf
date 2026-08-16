# Compliant: trail enables both API call rate and API error rate insights.
resource "aws_cloudtrail" "pass_example" {
  name           = "example"
  s3_bucket_name = "example-bucket"

  insight_selector {
    insight_type = "ApiCallRateInsight"
  }

  insight_selector {
    insight_type = "ApiErrorRateInsight"
  }
}
