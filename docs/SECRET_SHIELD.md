# Secret Leak Shield Specification

HookCraft includes an integrated real-time secret scanner that evaluates staged diff hunks prior to commit creation.

---

## 1. Detection Heuristics

The scanner operates in two phases for each added line in `git diff --staged -U0`:

### Phase 1: Pre-Compiled Provider Signatures
Lines are evaluated against regular expression signatures representing known vendor credential formats. Signatures include:
- Amazon Web Services: IAM Access Key ID (`AKIA`, `ABIA`, `ACCA`, `ASIA`), Secret Access Keys.
- GitHub: Classic Personal Access Tokens (`ghp_`), Fine-Grained PATs (`github_pat_`), OAuth Tokens (`gho_`).
- GitLab: Personal Access Tokens (`glpat-`).
- OpenAI: Project API Keys (`sk-proj-`), User Keys (`sk-`).
- Anthropic: Claude API Keys (`sk-ant-`).
- Stripe: Live Secret Keys (`sk_live_`), Restricted Keys (`rk_live_`).
- Slack: Bot Tokens (`xoxb-`), User Tokens (`xoxp-`), Incoming Webhook URLs (`hooks.slack.com`).
- Google Cloud: API Keys (`AIzaSy...`).
- Hugging Face: User Access Tokens (`hf_`).
- SendGrid: API Keys (`SG.`).
- Twilio: API Keys (`SK...`).
- Supabase: Service Role JWT secrets.
- Cryptographic Keys: Unencrypted and encrypted PEM headers (`-----BEGIN RSA PRIVATE KEY-----`, OpenSSH, EC, DSA).

### Phase 2: Shannon Entropy Analysis
Unstructured candidate tokens longer than 20 characters are analyzed using Shannon entropy:

$$H(X) = -\sum_{i=1}^{n} P(x_i) \log_2 P(x_i)$$

Where $P(x_i)$ is the probability of character $x_i$ occurring in token $X$.

- Typical English source code: 2.0 to 3.2 bits per byte.
- Base64 encoded hashes and cryptographic tokens: 3.8 to 5.2 bits per byte.

If a token's entropy meets or exceeds `settings.entropy_threshold` (default 3.8), HookCraft flags the token as a potential secret.

---

## 2. False Positive Prevention

HookCraft applies multiple exclusion layers to avoid blocking valid code:

1. Excluded files: Lockfiles (`package-lock.json`, `go.sum`, `Cargo.lock`, `pnpm-lock.yaml`) and media assets are bypassed automatically.
2. Character patterns: Standard UUIDs (`xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`), continuous hexadecimal commit SHAs, and URLs (`http://`, `https://`) are exempt from entropy testing.
3. Common mock words: Tokens containing substrings like `EXAMPLE`, `TEST`, `PLACEHOLDER`, or `DUMMY` are ignored.

---

## 3. Inline Suppression

To deliberately commit an example token (such as in a test fixture or documentation), add one of the following suppression comments to the line:

```go
const TestKey = "AKIA1234567890EXAMPLE" // hookcraft:ignore
```

Supported comments:
- `// hookcraft:ignore`
- `// hookcraft:allow`
- `# hookcraft:ignore`
- `// nosec`
- `pragma: allowlist secret`
