resource "aws_emr_cluster" "analytics" {
  name          = "analytics"
  release_label = "emr-7.1.0"
  service_role  = "EMR_DefaultRole"
}
