# Non-compliant: no ec2_attributes block, so the cluster is not pinned to a
# private subnet and defaults to public accessibility.
resource "aws_emr_cluster" "fail_example" {
  name          = "no-ec2attrs-cluster"
  release_label = "emr-6.10.0"
  service_role  = "EMR_DefaultRole"
}
