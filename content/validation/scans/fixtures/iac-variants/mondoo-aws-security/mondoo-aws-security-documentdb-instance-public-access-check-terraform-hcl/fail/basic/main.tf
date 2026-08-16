# Non-compliant: DocumentDB instance is publicly accessible.
resource "aws_docdb_cluster_instance" "fail_example" {
  identifier          = "fail-example"
  cluster_identifier  = "my-cluster"
  instance_class      = "db.r5.large"
  publicly_accessible = true
}
