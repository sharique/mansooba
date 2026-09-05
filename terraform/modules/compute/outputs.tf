output "public_ip" {
  description = "Auto-assigned public IP of the EC2 instance. Changes on stop/start — see the comment above aws_instance.app in main.tf."
  value       = aws_instance.app.public_ip
}

output "ssh_command" {
  description = "Command to SSH into the EC2 instance."
  value       = "ssh -i ~/.ssh/mansooba ec2-user@${aws_instance.app.public_ip}"
}
