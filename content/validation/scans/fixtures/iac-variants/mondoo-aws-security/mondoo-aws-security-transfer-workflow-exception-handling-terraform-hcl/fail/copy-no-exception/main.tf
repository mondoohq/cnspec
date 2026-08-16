resource "aws_transfer_workflow" "example" {
  steps {
    type = "COPY"
    copy_step_details {
      name = "copy-archive"
      destination_file_location {
        s3_file_location {
          bucket = aws_s3_bucket.archive.id
          key    = "archive/"
        }
      }
    }
  }
}
