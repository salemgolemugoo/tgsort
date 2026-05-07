terraform {
  source = "git::https://example.com/module.git"
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

# Networking layer
dependency "vpc" {
  config_path = "../vpc"
}

dependency "z_other" {
  config_path = "../z-other"
}

inputs = {
  a_var = "aaa"
  m_var = "mmm"
  z_var = "zzz"
}
