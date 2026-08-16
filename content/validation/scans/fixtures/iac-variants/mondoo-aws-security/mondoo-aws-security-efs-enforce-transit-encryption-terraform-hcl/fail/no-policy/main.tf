# Encryption at rest is on, but nothing requires clients to mount over TLS, so
# NFS traffic can cross the VPC in cleartext.
resource "aws_efs_file_system" "shared" {
  creation_token = "shared-app-data"
  encrypted      = true
}
