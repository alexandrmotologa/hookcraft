package secrets

import (
	"regexp"
)

// Rule defines a secret pattern signature.
type Rule struct {
	ID          string
	Name        string
	Category    string // "cloud", "saas", "crypto", "auth"
	Pattern     *regexp.Regexp
	Keywords    []string // Fast preliminary substring check before regex
	MinEntropy  float64
	Description string
}

// SecretRules holds the comprehensive rule registry for 25+ providers.
var SecretRules = []Rule{
	{
		ID:          "aws-access-key-id",
		Name:        "AWS Access Key ID",
		Category:    "cloud",
		Pattern:     regexp.MustCompile(`\b(AKIA|ABIA|ACCA|ASIA)[0-9A-Z]{16}\b`),
		Keywords:    []string{"AKIA", "ABIA", "ACCA", "ASIA"},
		Description: "AWS IAM Access Key identifier",
	},
	{
		ID:          "aws-secret-access-key",
		Name:        "AWS Secret Access Key",
		Category:    "cloud",
		Pattern:     regexp.MustCompile(`(?i)(?:aws_secret_access_key|aws_secret|aws_key)\s*[:=]\s*['"]?([A-Za-z0-9/+=]{40})['"]?`),
		Keywords:    []string{"aws"},
		MinEntropy:  4.2,
		Description: "AWS Secret Access Key paired with IAM user",
	},
	{
		ID:          "github-personal-access-token",
		Name:        "GitHub Personal Access Token",
		Category:    "saas",
		Pattern:     regexp.MustCompile(`\bghp_[0-9a-zA-Z]{34,40}\b`),
		Keywords:    []string{"ghp_"},
		Description: "GitHub classic personal access token",
	},
	{
		ID:          "github-fine-grained-pat",
		Name:        "GitHub Fine-Grained PAT",
		Category:    "saas",
		Pattern:     regexp.MustCompile(`\bgithub_pat_[0-9a-zA-Z_]{82}\b`),
		Keywords:    []string{"github_pat_"},
		Description: "GitHub modern fine-grained token",
	},
	{
		ID:          "github-oauth-token",
		Name:        "GitHub OAuth Access Token",
		Category:    "saas",
		Pattern:     regexp.MustCompile(`\bgho_[0-9a-zA-Z]{36}\b`),
		Keywords:    []string{"gho_"},
		Description: "GitHub OAuth client access token",
	},
	{
		ID:          "gitlab-pat",
		Name:        "GitLab Personal Access Token",
		Category:    "saas",
		Pattern:     regexp.MustCompile(`\bglpat-[0-9a-zA-Z\-_]{20,}\b`),
		Keywords:    []string{"glpat-"},
		Description: "GitLab user personal token",
	},
	{
		ID:          "openai-api-key",
		Name:        "OpenAI API Key",
		Category:    "saas",
		Pattern:     regexp.MustCompile(`\b(sk-[a-zA-Z0-9]{20,48}|sk-proj-[a-zA-Z0-9_\-]{48,128})\b`),
		Keywords:    []string{"sk-"},
		Description: "OpenAI developer platform API key",
	},
	{
		ID:          "anthropic-api-key",
		Name:        "Anthropic API Key",
		Category:    "saas",
		Pattern:     regexp.MustCompile(`\bsk-ant-[a-zA-Z0-9_\-]{32,}\b`),
		Keywords:    []string{"sk-ant-"},
		Description: "Anthropic Claude API key",
	},
	{
		ID:          "stripe-secret-key",
		Name:        "Stripe Secret Key",
		Category:    "saas",
		Pattern:     regexp.MustCompile(`\b(?:sk|rk)_live_[0-9a-zA-Z]{24,}\b`),
		Keywords:    []string{"sk_live_", "rk_live_"},
		Description: "Stripe live production secret key",
	},
	{
		ID:          "slack-bot-token",
		Name:        "Slack Bot Token",
		Category:    "saas",
		Pattern:     regexp.MustCompile(`\bxoxb-[0-9]{10,13}-[0-9]{10,13}-[a-zA-Z0-9]{24}\b`),
		Keywords:    []string{"xoxb-"},
		Description: "Slack bot authentication token",
	},
	{
		ID:          "slack-user-token",
		Name:        "Slack User Token",
		Category:    "saas",
		Pattern:     regexp.MustCompile(`\bxoxp-[0-9]{10,13}-[0-9]{10,13}-[a-zA-Z0-9]{24,32}\b`),
		Keywords:    []string{"xoxp-"},
		Description: "Slack user personal workspace token",
	},
	{
		ID:          "slack-webhook-url",
		Name:        "Slack Incoming Webhook URL",
		Category:    "saas",
		Pattern:     regexp.MustCompile(`https://hooks\.slack\.com/services/T[a-zA-Z0-9_]+/B[a-zA-Z0-9_]+/[a-zA-Z0-9_]+`),
		Keywords:    []string{"hooks.slack.com"},
		Description: "Directly accessible Slack incoming webhook",
	},
	{
		ID:          "google-api-key",
		Name:        "Google Cloud API Key",
		Category:    "cloud",
		Pattern:     regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{30,40}\b`),
		Keywords:    []string{"AIza"},
		Description: "Google Cloud Platform API identifier",
	},
	{
		ID:          "huggingface-token",
		Name:        "HuggingFace Access Token",
		Category:    "saas",
		Pattern:     regexp.MustCompile(`\bhf_[a-zA-Z0-9]{34,}\b`),
		Keywords:    []string{"hf_"},
		Description: "Hugging Face model repository token",
	},
	{
		ID:          "sendgrid-api-key",
		Name:        "SendGrid API Key",
		Category:    "saas",
		Pattern:     regexp.MustCompile(`\bSG\.[a-zA-Z0-9_\-]{22}\.[a-zA-Z0-9_\-]{43}\b`),
		Keywords:    []string{"SG."},
		Description: "SendGrid email delivery API credential",
	},
	{
		ID:          "twilio-api-key",
		Name:        "Twilio API Key",
		Category:    "saas",
		Pattern:     regexp.MustCompile(`\bSK[0-9a-fA-F]{32}\b`),
		Keywords:    []string{"SK"},
		Description: "Twilio communications API secret",
	},
	{
		ID:          "private-key-header",
		Name:        "Private Cryptographic Key",
		Category:    "crypto",
		Pattern:     regexp.MustCompile(`-----BEGIN (?:[A-Z0-9_-]+ )?PRIVATE KEY-----`),
		Keywords:    []string{"-----BEGIN"},
		Description: "Unencrypted or encrypted private cryptographic key header",
	},
	{
		ID:          "supabase-service-role",
		Name:        "Supabase Service Role Key",
		Category:    "saas",
		Pattern:     regexp.MustCompile(`(?i)(?:supabase.*service.*role|SUPABASE_SERVICE_KEY)\s*[:=]\s*['"]?([a-zA-Z0-9_\-\.]{50,})['"]?`),
		Keywords:    []string{"supabase"},
		Description: "Supabase administrative service role JWT key",
	},
	{
		ID:          "generic-api-key-assignment",
		Name:        "Generic High-Entropy API Secret Assignment",
		Category:    "auth",
		Pattern:     regexp.MustCompile(`(?i)(?:api_key|apikey|secret_key|auth_token|access_token|private_token)\s*[:=]\s*['"]([a-zA-Z0-9_\-]{24,})['"]`),
		Keywords:    []string{"api_key", "apikey", "secret_key", "auth_token", "access_token"},
		MinEntropy:  3.9,
		Description: "High-entropy variable assignment with secret keyword",
	},
	{
		ID:          "password-hardcoded-assignment",
		Name:        "Hardcoded Password Assignment",
		Category:    "auth",
		Pattern:     regexp.MustCompile(`(?i)(?:password|passwd|db_password)\s*[:=]\s*['"]([^'"\s]{10,})['"]`),
		Keywords:    []string{"password", "passwd", "db_password"},
		MinEntropy:  3.6,
		Description: "Inline hardcoded password string in source code",
	},
}
