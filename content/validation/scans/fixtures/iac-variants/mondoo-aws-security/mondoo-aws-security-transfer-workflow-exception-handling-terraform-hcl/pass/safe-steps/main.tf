resource "aws_transfer_workflow" "example" {
  steps {
    type = "TAG"
    tag_step_details {
      name = "tag-processed"
      tags {
        key   = "status"
        value = "processed"
      }
    }
  }
}
