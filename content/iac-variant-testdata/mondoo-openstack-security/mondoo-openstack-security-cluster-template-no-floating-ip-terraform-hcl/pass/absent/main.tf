resource "openstack_containerinfra_clustertemplate_v1" "k8s" {
  name           = "k8s-cluster-template"
  image          = "fedora-coreos-latest"
  coe            = "kubernetes"
  flavor         = "m1.medium"
  master_flavor  = "m1.medium"
  dns_nameserver = "1.1.1.1"
}
