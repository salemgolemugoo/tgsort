# Configures networking
dependency "vpc" {
  config_path = "../vpc"
}

# Sets up k8s
dependency "eks" {
  config_path = "../eks"
}
