# Compliant: notebook instance sets a KMS key.
resource "aws_sagemaker_notebook_instance" "pass_example" {
  name          = "example-notebook"
  instance_type = "ml.t3.medium"
  role_arn      = "arn:aws:iam::123456789012:role/example"
  kms_key_id    = "arn:aws:kms:us-east-1:123456789012:key/abcd1234-a123-456a-a12b-a123b4cd56ef"
}
