# Compliant: DocumentDB instance is not publicly accessible.
resource "aws_docdb_cluster_instance" "pass_example" {
  identifier          = "pass-example"
  cluster_identifier  = "my-cluster"
  instance_class      = "db.r5.large"
  publicly_accessible = false
}
