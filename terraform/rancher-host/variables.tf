variable "region" {
  description = "AWS region for the Rancher host"
  type        = string
  default     = "ap-southeast-1"
}

variable "name" {
  description = "Name tag / resource name prefix for the Rancher host"
  type        = string
  default     = "rancher-host"
}

variable "instance_type" {
  description = "EC2 instance type — t3.micro (1GB RAM) was not enough for k3s + Rancher together (observed OOM-level unresponsiveness); t3.medium (4GB) matches Rancher's own documented minimum"
  type        = string
  default     = "t3.medium"
}

variable "tags" {
  description = "Tags applied to every resource this config creates"
  type        = map(string)
  default = {
    Project   = "social-chat-app"
    ManagedBy = "terraform"
    Component = "rancher-host"
  }
}
