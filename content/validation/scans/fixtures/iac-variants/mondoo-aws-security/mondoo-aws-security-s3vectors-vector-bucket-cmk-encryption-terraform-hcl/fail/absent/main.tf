# No encryption configuration, so vectors use Amazon S3 managed keys.
resource "aws_s3vectors_vector_bucket" "example" {
  vector_bucket_name = "rag-vectors"
}
