package risk

// statefulResourceTypes are resources whose replacement typically causes
// data loss, downtime, or loss of durable state.
var statefulResourceTypes = []string{
	"aws_db_instance", "aws_rds_cluster", "aws_rds_cluster_instance",
	"google_sql_database_instance", "azurerm_postgresql_server",
	"azurerm_mysql_server", "azurerm_sql_database",
	"aws_s3_bucket", "google_storage_bucket", "azurerm_storage_account",
	"aws_sqs_queue", "aws_sns_topic", "google_pubsub_subscription",
	"aws_elasticache_cluster", "aws_elasticache_replication_group",
	"aws_efs_file_system", "aws_dynamodb_table",
	"aws_lb", "aws_elb",
	"kubernetes_persistent_volume", "kubernetes_persistent_volume_claim",
	"kubernetes_stateful_set",
}

// securitySensitiveResourceTypes are resources where any change warrants
// heightened scrutiny because they affect access control or network
// exposure.
var securitySensitiveResourceTypes = []string{
	"aws_iam_role", "aws_iam_policy", "aws_iam_user", "aws_iam_role_policy",
	"aws_iam_user_policy", "aws_iam_group_policy", "aws_iam_group_policy_attachment",
	"aws_iam_role_policy_attachment", "aws_iam_user_policy_attachment",
	"google_project_iam_binding", "google_project_iam_member", "google_project_iam_policy",
	"azurerm_role_assignment", "azurerm_role_definition",
	"aws_security_group", "aws_security_group_rule",
	"aws_network_acl", "aws_network_acl_rule",
	"google_compute_firewall", "azurerm_network_security_rule", "azurerm_network_security_group",
	"aws_eip", "aws_eip_association", "google_compute_address",
	"aws_s3_bucket_public_access_block",
}

// DefaultRules returns tfsum's built-in deterministic risk rules,
// implementing the default rule set described in the project's risk
// engine requirements.
func DefaultRules() []Rule {
	return []Rule{
		{
			ID:       "resource-deletion",
			Match:    RuleMatch{Actions: []string{"delete"}},
			Severity: "critical",
			Message:  "Resource will be destroyed.",
		},
		{
			ID:       "resource-replacement",
			Match:    RuleMatch{Actions: []string{"delete", "create"}},
			Severity: "high",
			Message:  "Resource will be replaced (destroy then create).",
		},
		{
			ID:       "stateful-replacement",
			Match:    RuleMatch{ResourceTypes: statefulResourceTypes, Actions: []string{"delete", "create"}},
			Severity: "critical",
			Message:  "Replacement of a stateful resource may cause downtime or data loss.",
		},
		{
			ID:       "security-sensitive-change",
			Match:    RuleMatch{ResourceTypes: securitySensitiveResourceTypes},
			Severity: "high",
			Message:  "Change affects a security-sensitive resource.",
		},
		{
			ID:       "public-ingress-ipv4",
			Match:    RuleMatch{Attribute: "cidr_blocks", Contains: "0.0.0.0/0"},
			Severity: "critical",
			Message:  "Resource allows inbound traffic from the public internet (0.0.0.0/0).",
		},
		{
			ID:       "public-ingress-ipv6",
			Match:    RuleMatch{Attribute: "ipv6_cidr_blocks", Contains: "::/0"},
			Severity: "critical",
			Message:  "Resource allows inbound traffic from the public internet (::/0).",
		},
		{
			ID:       "iam-wildcard-action",
			Match:    RuleMatch{Attribute: "actions", Contains: "*"},
			Severity: "critical",
			Message:  "IAM policy change grants a wildcard (*) action.",
		},
		{
			ID:       "tag-only-change",
			Match:    RuleMatch{ChangedAttributes: []string{"tags", "tags_all"}},
			Severity: "low",
			Message:  "Only tags changed.",
		},
		{
			ID:       "unknown-values",
			Match:    RuleMatch{Unknown: true},
			Severity: "medium",
			Message:  "Resource contains values that will only be known after apply.",
		},
	}
}
