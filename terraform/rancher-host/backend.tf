terraform {
  backend "s3" {
    bucket       = "social-chat-app-tfstate-43a40f5b" # same bucket terraform/backend.tf uses
    key          = "social-chat-app/rancher-host/terraform.tfstate"
    region       = "ap-southeast-1"
    encrypt      = true
    use_lockfile = true
  }
}
