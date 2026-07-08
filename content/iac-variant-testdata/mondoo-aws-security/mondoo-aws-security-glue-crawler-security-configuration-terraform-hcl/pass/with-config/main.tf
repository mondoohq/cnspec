# Compliant: Glue crawler references a security configuration.
resource "aws_glue_crawler" "pass_example" {
  name                   = "example-crawler"
  role                   = "arn:aws:iam::111122223333:role/glue-crawler"
  database_name          = "example_db"
  security_configuration = "example-security-config"

  s3_target {
    path = "s3://example-bucket/data/"
  }
}
