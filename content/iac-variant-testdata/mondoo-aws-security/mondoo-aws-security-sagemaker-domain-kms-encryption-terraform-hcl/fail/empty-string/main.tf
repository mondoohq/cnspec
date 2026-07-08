# Non-compliant: kms_key_id resolves to an empty string.
resource "aws_sagemaker_domain" "fail_example" {
  domain_name = "example-domain"
  auth_mode   = "IAM"
  vpc_id      = "vpc-0123456789abcdef0"
  subnet_ids  = ["subnet-0123456789abcdef0"]
  kms_key_id  = ""

  default_user_settings {
    execution_role = "arn:aws:iam::123456789012:role/example"
  }
}
