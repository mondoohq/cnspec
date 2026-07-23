# Non-compliant: EMR Studio has no encryption_key_arn.
resource "aws_emr_studio" "fail_example" {
  auth_mode                   = "IAM"
  default_s3_location         = "s3://example-bucket/studio/"
  engine_security_group_id    = "sg-0123456789abcdef0"
  name                        = "example-studio"
  service_role                = "arn:aws:iam::111122223333:role/emr-studio"
  subnet_ids                  = ["subnet-0123456789abcdef0"]
  vpc_id                      = "vpc-0123456789abcdef0"
  workspace_security_group_id = "sg-0123456789abcdef1"
}
