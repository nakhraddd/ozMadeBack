terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
  backend "gcs" {
    # Replace with your own bucket name for storing Terraform state
    bucket = "ozmadeback-tf-state-bucket"
    prefix = "terraform/state"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# Reserve a static IP address
resource "google_compute_address" "static_ip" {
  name   = "ozmadeback-static-ip"
  region = var.region
}

# Define the firewall rule to allow necessary traffic
resource "google_compute_firewall" "allow_app_traffic" {
  name    = "allow-ozmadeback-traffic"
  network = "default"

  allow {
    protocol = "tcp"
    # 22 (SSH), 80 (HTTP), 443 (HTTPS), 8080 (Go App)
    ports = ["22", "80", "443", "8080"]
  }
  source_ranges = ["0.0.0.0/0"]
}

# Create the Compute Engine VM
resource "google_compute_instance" "app_server" {
  name         = "ozmadeback-vm"
  machine_type = "e2-medium" # e2-medium is a cost-effective choice
  zone         = "${var.region}-a"

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
      size  = 20 # Start with a smaller disk size
    }
  }

  network_interface {
    network = "default"
    access_config {
      nat_ip = google_compute_address.static_ip.address
    }
  }

  metadata = {
    "ssh-keys" = "gcp-user:${var.ssh_public_key}"
  }

  # Allow the instance to access Cloud SQL and other GCP services
  service_account {
    email  = "default"
    scopes = ["cloud-platform"]
  }
}