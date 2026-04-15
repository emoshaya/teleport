module "aws_discovery" {
  source = "../.."

  teleport_proxy_public_addr    = "example.teleport.sh:443"
  teleport_discovery_group_name = "cloud-discovery-group"

  # Enroll resources from all AWS Accounts in the Organization
  # Only EC2 resource discovery is supported for organization-wide discovery.
  enroll_organization_accounts = true

  # Discover EC2 instances with matching rules
  aws_matchers = [
    {
      types   = ["ec2"]
      regions = ["*"]
      tags = {
        env = ["prod"]
      }
    },
  ]

  # Apply the additional Teleport label "origin=example" to all Teleport resources created by this module
  apply_teleport_resource_labels = { origin = "example" }
  # Apply the additional AWS tag "origin=example" to all AWS resources created by this module
  apply_aws_tags = { origin = "example" }

  # Example of specifying the IAM role name to assume in child accounts.
  # This role must be created manually in each child account with the appropriate trust relationship and permissions for discovering resources.
  # Check the module outputs for the required trust relationship and permissions.
  aws_iam_role_name_for_child_accounts = "teleport-discovery-child-account-role"
}
