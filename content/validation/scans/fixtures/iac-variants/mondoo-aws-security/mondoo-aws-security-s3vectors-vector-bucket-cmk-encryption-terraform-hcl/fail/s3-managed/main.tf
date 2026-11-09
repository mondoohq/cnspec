# Vectors use Amazon S3 managed keys.
resource "aws_s3vectors_vector_bucket" "example" {
  vector_bucket_name = "rag-vectors"

  encryption_configuration {
    sse_type = "AES256"
  }
}
