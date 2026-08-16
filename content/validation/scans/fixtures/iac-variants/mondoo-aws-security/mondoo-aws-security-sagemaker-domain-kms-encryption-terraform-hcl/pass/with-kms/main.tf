# Compliant: SageMaker domain sets a KMS key.
resource "aws_sagemaker_domain" "pass_example" {
  domain_name = "example-domain"
  auth_mode   = "IAM"
  vpc_id      = "vpc-0123456789abcdef0"
  subnet_ids  = ["subnet-0123456789abcdef0"]
  kms_key_id  = "arn:aws:kms:us-east-1:123456789012:key/abcd1234-a123-456a-a12b-a123b4cd56ef"

  default_user_settings {
    execution_role = "arn:aws:iam::123456789012:role/example"
  }
}
