resource "google_service_account" "default" {
  account_id   = "service-account-id"
  display_name = "Service Account"
}

resource "google_container_cluster" "primary" {
  name     = "my-gke-cluster"
  location = "us-central1"

  # We can't create a cluster with no node pool defined, but we want to only use
  # separately managed node pools. So we create the smallest possible default
  # node pool and immediately delete it.
  remove_default_node_pool = true
  initial_node_count       = 1
  monitoring_service = "monitoring.googleapis.com/kubernetes"
  logging_service = "logging.googleapis.com/kubernetes"
  ip_allocation_policy     {
    cluster_secondary_range_name  = "some range name"
    services_secondary_range_name = "some range name"
  }
  master_authorized_networks_config = [{
    cidr_blocks = [{
      cidr_block = "10.10.128.0/24"
      display_name = "internal"
    }]
  }]
  network_policy {
    enabled = true
  }
  resource_labels = {
    "env" = "staging"
  }
}

resource "google_container_node_pool" "good_example" {
  name       = "my-node-pool"
  cluster    = google_container_cluster.primary.id
  node_count = 1

  node_config {
    preemptible  = true
    machine_type = "e2-medium"
    image_type = "COS_CONTAINERD"

    # Google recommends custom service accounts that have cloud-platform scope and permissions granted via IAM Roles.
    service_account = google_service_account.default.email
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }
  management {
    auto_repair = true
  }
}

resource "google_service_account" "default" {
  account_id   = "service-account-id"
  display_name = "Service Account"
}

resource "google_container_cluster" "primary" {
  name     = "my-gke-cluster"
  location = "us-central1"

  # We can't create a cluster with no node pool defined, but we want to only use
  # separately managed node pools. So we create the smallest possible default
  # node pool and immediately delete it.
  remove_default_node_pool = true
  initial_node_count       = 1
  logging_service = "logging.googleapis.com/kubernetes"
  monitoring_service = "monitoring.googleapis.com/kubernetes"
  ip_allocation_policy     {
    cluster_secondary_range_name  = "some range name"
    services_secondary_range_name = "some range name"
  }
  master_authorized_networks_config = [{
    cidr_blocks = [{
      cidr_block = "10.10.128.0/24"
      display_name = "internal"
    }]
  }]
  network_policy {
    enabled = true
  }
  resource_labels = {
    "env" = "staging"
  }
}

resource "google_container_node_pool" "good_example" {
  name       = "my-node-pool"
  cluster    = google_container_cluster.primary.id
  node_count = 1

  node_config {
    preemptible  = true
    machine_type = "e2-medium"
    image_type = "COS_CONTAINERD"

    # Google recommends custom service accounts that have cloud-platform scope and permissions granted via IAM Roles.
    service_account = google_service_account.default.email
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }
  management {
    auto_upgrade = true
  }
}

resource "google_service_account" "default" {
  account_id   = "service-account-id"
  display_name = "Service Account"
}

resource "google_container_cluster" "good_example" {
  name     = "my-gke-cluster"
  location = "us-central1"

  # We can't create a cluster with no node pool defined, but we want to only use
  # separately managed node pools. So we create the smallest possible default
  # node pool and immediately delete it.
  remove_default_node_pool = true
  initial_node_count       = 1
  logging_service = "logging.googleapis.com/kubernetes"
  monitoring_service = "monitoring.googleapis.com/kubernetes"
  ip_allocation_policy     {
    cluster_secondary_range_name  = "some range name"
    services_secondary_range_name = "some range name"
  }
  master_authorized_networks_config = [{
    cidr_blocks = [{
      cidr_block = "10.10.128.0/24"
      display_name = "internal"
    }]
  }]
  network_policy {
    enabled = true
  }
  resource_labels = {
    "env" = "staging"
  }
}

resource "google_container_node_pool" "primary_preemptible_nodes" {
  name       = "my-node-pool"
  location   = "us-central1"
  cluster    = google_container_cluster.primary.name
  node_count = 1

  node_config {
    preemptible  = true
    machine_type = "e2-medium"
    image_type = "COS_CONTAINERD"

    # Google recommends custom service accounts that have cloud-platform scope and permissions granted via IAM Roles.
    service_account = google_service_account.default.email
    oauth_scopes    = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }
}

resource "google_service_account" "default" {
  account_id   = "service-account-id"
  display_name = "Service Account"
}

resource "google_container_cluster" "good_example" {
  name     = "my-gke-cluster"
  location = "us-central1"

  # We can't create a cluster with no node pool defined, but we want to only use
  # separately managed node pools. So we create the smallest possible default
  # node pool and immediately delete it.
  remove_default_node_pool = true
  initial_node_count       = 1
  logging_service = "logging.googleapis.com/kubernetes"
  monitoring_service = "monitoring.googleapis.com/kubernetes"
  ip_allocation_policy     {
    cluster_secondary_range_name  = "some range name"
    services_secondary_range_name = "some range name"
  }
  master_authorized_networks_config = [{
    cidr_blocks = [{
      cidr_block = "10.10.128.0/24"
      display_name = "internal"
    }]
  }]
  network_policy {
    enabled = true
  }
  resource_labels = {
    "env" = "staging"
  }
}

resource "google_container_node_pool" "primary_preemptible_nodes" {
  name       = "my-node-pool"
  location   = "us-central1"
  cluster    = google_container_cluster.primary.name
  node_count = 1

  node_config {
    preemptible  = true
    machine_type = "e2-medium"
    image_type = "COS_CONTAINERD"

    # Google recommends custom service accounts that have cloud-platform scope and permissions granted via IAM Roles.
    service_account = google_service_account.default.email
    oauth_scopes    = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }
}

resource "google_container_cluster" "good_example" {
  name     = "my-gke-cluster"
  location = "us-central1"

  # We can't create a cluster with no node pool defined, but we want to only use
  # separately managed node pools. So we create the smallest possible default
  # node pool and immediately delete it.
  remove_default_node_pool = true
  initial_node_count       = 1
  logging_service = "logging.googleapis.com/kubernetes"
  monitoring_service = "monitoring.googleapis.com/kubernetes"
  ip_allocation_policy     {
    cluster_secondary_range_name  = "some range name"
    services_secondary_range_name = "some range name"
  }
  master_authorized_networks_config = [{
    cidr_blocks = [{
      cidr_block = "10.10.128.0/24"
      display_name = "internal"
    }]
  }]
  network_policy {
    enabled = true
  }
  resource_labels = {
    "env" = "staging"
  }
}

resource "google_container_node_pool" "primary_preemptible_nodes" {
  name       = "my-node-pool"
  location   = "us-central1"
  cluster    = google_container_cluster.primary.name
  node_count = 1

  node_config {
    preemptible  = true
    machine_type = "e2-medium"
    image_type = "COS_CONTAINERD"

    # Google recommends custom service accounts that have cloud-platform scope and permissions granted via IAM Roles.
    service_account = google_service_account.default.email
    oauth_scopes    = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }
}

resource "google_container_cluster" "primary" {
  name     = "my-gke-cluster"
  location = "us-central1"

  # We can't create a cluster with no node pool defined, but we want to only use
  # separately managed node pools. So we create the smallest possible default
  # node pool and immediately delete it.
  remove_default_node_pool = true
  initial_node_count       = 1
  logging_service = "logging.googleapis.com/kubernetes"
  monitoring_service = "monitoring.googleapis.com/kubernetes"
  ip_allocation_policy     {
    cluster_secondary_range_name  = "some range name"
    services_secondary_range_name = "some range name"
  }
  master_authorized_networks_config = [{
    cidr_blocks = [{
      cidr_block = "10.10.128.0/24"
      display_name = "internal"
    }]
  }]

  metadata = {
    disable-legacy-endpoints = true
  }

  network_policy {
    enabled = true
  }
  resource_labels = {
    "env" = "staging"
  }
}

resource "google_container_cluster" "primary" {
  name     = "my-gke-cluster"
  location = "us-central1"

  # We can't create a cluster with no node pool defined, but we want to only use
  # separately managed node pools. So we create the smallest possible default
  # node pool and immediately delete it.
  remove_default_node_pool = true
  initial_node_count       = 1
  logging_service = "logging.googleapis.com/kubernetes"
  monitoring_service = "monitoring.googleapis.com/kubernetes"
  ip_allocation_policy     {
    cluster_secondary_range_name  = "some range name"
    services_secondary_range_name = "some range name"
  }
  master_authorized_networks_config = [{
    cidr_blocks = [{
      cidr_block = "10.10.128.0/24"
      display_name = "internal"
    }]
  }]

  metadata = {}

  network_policy {
    enabled = true
  }

  master_auth {
    username = ""
    password = "" 
    client_certificate_config {
      issue_client_certificate = false
    }
  }
  resource_labels = {
    "env" = "staging"
  }
}

resource "google_container_cluster" "master_authorized_networks_config" {
  name     = "my-gke-cluster"
  location = "us-central1"

  # We can't create a cluster with no node pool defined, but we want to only use
  # separately managed node pools. So we create the smallest possible default
  # node pool and immediately delete it.
  remove_default_node_pool = true
  initial_node_count       = 1
  logging_service = "logging.googleapis.com/kubernetes"
  monitoring_service = "monitoring.googleapis.com/kubernetes"
  ip_allocation_policy     {
    cluster_secondary_range_name  = "some range name"
    services_secondary_range_name = "some range name"
  }
  master_authorized_networks_config = [{
    cidr_blocks = [
      {
        cidr_block = "10.10.128.0/24"
        display_name = "internal-1"
      },
      {
        cidr_block = "10.10.129.0/24"
        display_name = "internal-2"
      },
    ]
  }]

  metadata = {}

  network_policy {
    enabled = true
  }

  master_auth {
    username = ""
    password = "" 
    client_certificate_config {
      issue_client_certificate = false
    }
  }
  resource_labels = {
    "env" = "staging"
  }
  enable_legacy_abac = false
}

resource "google_container_node_pool" "good_node_metadata_1" {
  node_config {
    image_type = "COS_CONTAINERD"
    service_account = google_service_account.default.email

    workload_metadata_config {
      node_metadata = "GKE_METADATA_SERVER"
    }
  }
}

resource "google_container_node_pool" "good_node_metadata_2" {
  node_config {
    image_type = "COS_CONTAINERD"
    service_account = google_service_account.default.email

    workload_metadata_config {
      node_metadata = "SECURE"
    }
  }
}

resource "google_container_node_pool" "good_example" {
  name       = "my-node-pool"
  cluster    = google_container_cluster.primary.id
  node_count = 1

  node_config {
    preemptible  = true
    machine_type = "e2-medium"

    # Google recommends custom service accounts that have cloud-platform scope and permissions granted via IAM Roles.
    service_account = google_service_account.default.email
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
    image_type = "COS_CONTAINERD"
  }
}
