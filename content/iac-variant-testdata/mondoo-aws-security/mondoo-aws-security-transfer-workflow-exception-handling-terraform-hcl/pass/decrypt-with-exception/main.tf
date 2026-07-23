resource "aws_transfer_workflow" "example" {
  steps {
    type = "DECRYPT"
    decrypt_step_details {
      name = "decrypt-incoming"
      type = "PGP"
      destination_file_location {
        s3_file_location {
          bucket = aws_s3_bucket.decrypted.id
          key    = "decrypted/"
        }
      }
    }
  }

  on_exception_steps {
    type = "TAG"
    tag_step_details {
      name = "tag-failed"
      tags {
        key   = "status"
        value = "failed"
      }
    }
  }
}
