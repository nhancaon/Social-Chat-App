output "public_ip" {
  description = "Stable Elastic IP — this is the Rancher URL: https://<this>"
  value       = aws_eip.rancher_host.public_ip
}

output "rancher_bootstrap_password" {
  description = "Pass this as bootstrapPassword when installing Rancher (helm --set or a values file)"
  value       = random_password.rancher_bootstrap.result
  sensitive   = true
}

output "ssm_connect_command" {
  description = "Run this to get a shell on the instance — no SSH key needed"
  value       = "aws ssm start-session --target ${aws_instance.rancher_host.id} --region ${var.region}"
}
