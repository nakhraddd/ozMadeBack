variable "project_id" {
  description = "The GCP project ID to deploy resources into."
  type        = string
}

variable "region" {
  description = "The GCP region for the resources."
  type        = string
  default     = "europe-west4"
}

variable "ssh_public_key" {
  description = "The SSH public key to be added to the VM for access."
  type        = string
  sensitive   = true
}

variable "db_password" {
  description = "The password for the Cloud SQL database user."
  type        = string
  sensitive   = true
}