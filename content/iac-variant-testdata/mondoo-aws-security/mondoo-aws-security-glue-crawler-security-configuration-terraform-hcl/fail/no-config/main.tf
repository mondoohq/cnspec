# Non-compliant: Glue crawler has no security_configuration set.
resource "aws_glue_crawler" "fail_example" {
  name          = "example-crawler"
  role          = "arn:aws:iam::111122223333:role/glue-crawler"
  database_name = "example_db"

  s3_target {
    path = "s3://example-bucket/data/"
  }
}
