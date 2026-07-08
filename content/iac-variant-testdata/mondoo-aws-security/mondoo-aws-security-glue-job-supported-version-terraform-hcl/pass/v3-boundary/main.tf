resource "aws_glue_job" "example" {
  name         = "example-job"
  role_arn     = "arn:aws:iam::123456789012:role/glue"
  glue_version = "3.0"

  command {
    script_location = "s3://my-bucket/my-script.py"
  }
}
