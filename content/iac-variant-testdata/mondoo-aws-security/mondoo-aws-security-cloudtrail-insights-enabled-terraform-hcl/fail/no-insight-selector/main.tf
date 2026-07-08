# Non-compliant: trail defines no insight_selector blocks at all, so neither
# API call rate nor API error rate insights are enabled.
resource "aws_cloudtrail" "fail_example" {
  name           = "example"
  s3_bucket_name = "example-bucket"

  event_selector {
    read_write_type           = "All"
    include_management_events = true
  }
}
