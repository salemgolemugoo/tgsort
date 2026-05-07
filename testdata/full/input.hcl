dependency "z_other" {
  config_path = "../z-other"
}

inputs = {
  z_var = "zzz"
  a_var = "aaa"
  m_var = "mmm"
}

locals {
  region = "us-east-1"
}

# Sets up k8s cluster
dependency "eks" {
  config_path = "../eks"

  mock_outputs = {
    cluster_name = "mock"
  }
}

terraform {
  source = "git::https://example.com/module.git"
}

# Networking layer
dependency "vpc" {
  config_path = "../vpc"
}
