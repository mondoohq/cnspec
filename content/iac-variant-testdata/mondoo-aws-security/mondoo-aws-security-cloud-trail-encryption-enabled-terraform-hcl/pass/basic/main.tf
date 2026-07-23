# Compliant: CloudTrail encrypts logs with a KMS key.
resource "aws_cloudtrail" "pass_example" {
  name           = "example-trail"
  s3_bucket_name = "example-bucket"
  kms_key_id     = "arn:aws:kms:us-east-1:111122223333:key/abcd"
}
