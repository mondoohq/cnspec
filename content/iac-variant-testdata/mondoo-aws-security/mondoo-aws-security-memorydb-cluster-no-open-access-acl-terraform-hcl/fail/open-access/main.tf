# "open-access" is the built-in ACL that permits any user with any password to
# run any command, which defeats MemoryDB's authentication entirely.
resource "aws_memorydb_cluster" "sessions" {
  name              = "sessions"
  acl_name          = "open-access"
  node_type         = "db.t4g.small"
  num_shards        = 2
  subnet_group_name = aws_memorydb_subnet_group.private.name
}
