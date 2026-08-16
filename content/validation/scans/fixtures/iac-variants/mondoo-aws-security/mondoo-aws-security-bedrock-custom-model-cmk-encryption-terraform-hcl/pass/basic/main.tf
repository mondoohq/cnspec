# Compliant: custom model encrypted with a customer-managed KMS key.
resource "aws_bedrock_custom_model" "pass_example" {
  custom_model_name     = "example-model"
  job_name              = "example-job"
  base_model_identifier = "arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-text-express-v1"
  role_arn              = "arn:aws:iam::111122223333:role/example"

  custom_model_kms_key_id = "arn:aws:kms:us-east-1:111122223333:key/abcd"
}
