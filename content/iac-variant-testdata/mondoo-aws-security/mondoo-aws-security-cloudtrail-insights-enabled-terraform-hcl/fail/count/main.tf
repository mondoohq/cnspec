# Non-compliant: counted trails enable only the API call rate insight.
resource "aws_cloudtrail" "fleet" {
  count          = 2
  name           = "fleet-${count.index}"
  s3_bucket_name = "example-bucket"
  insight_selector {
    insight_type = "ApiCallRateInsight"
  }
}
