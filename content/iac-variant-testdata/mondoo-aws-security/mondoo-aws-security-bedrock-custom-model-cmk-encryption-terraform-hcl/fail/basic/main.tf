# Non-compliant: custom model has no customer-managed KMS key configured.
resource "aws_bedrock_custom_model" "fail_example" {
  custom_model_name     = "example-model"
  job_name              = "example-job"
  base_model_identifier = "arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-text-express-v1"
  role_arn              = "arn:aws:iam::111122223333:role/example"
}
