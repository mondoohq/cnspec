# Compliant: the check only targets multi-region trails. A single-region trail
# without CloudWatch Logs integration is out of scope and must not be flagged.
resource "aws_cloudtrail" "pass_example" {
  name                  = "regional-example"
  s3_bucket_name        = "example-bucket"
  is_multi_region_trail = false
}
