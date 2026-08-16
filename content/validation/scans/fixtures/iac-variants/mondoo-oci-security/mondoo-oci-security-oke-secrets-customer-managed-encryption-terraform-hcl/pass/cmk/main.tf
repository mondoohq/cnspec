resource "oci_containerengine_cluster" "prod" {
  compartment_id     = var.compartment_ocid
  kubernetes_version = "v1.31.1"
  name               = "prod-oke"
  vcn_id             = oci_core_vcn.oke.id

  # Envelope-encrypt Kubernetes secrets with a customer-managed Vault key.
  kms_key_id = "ocid1.key.oc1.iad.bbqxxxxxxxxxx.abuwcljrxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

  options {
    service_lb_subnet_ids = [oci_core_subnet.lb.id]

    kubernetes_network_config {
      pods_cidr     = "10.244.0.0/16"
      services_cidr = "10.96.0.0/16"
    }
  }
}
