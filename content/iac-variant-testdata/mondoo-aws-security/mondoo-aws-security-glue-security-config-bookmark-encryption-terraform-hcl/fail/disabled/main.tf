resource "aws_glue_security_configuration" "example" {
  name = "example-security-config"

  encryption_configuration {
    job_bookmarks_encryption {
      job_bookmarks_encryption_mode = "DISABLED"
    }
  }
}
