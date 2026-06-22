# Season Restart Guide — Redeploying Bandits Notification

This document explains how to bring the Bandits Notification service back online after it
was shut down at the end of a season. Follow it top-to-bottom; it is written to be
self-contained so you don't need to remember any prior state.

## Why this guide exists

At the end of the 2025–2026 season the AWS deployment was torn down to stop ongoing costs.
**The SOPS/KMS key was deleted**, which means the encrypted `secrets.yaml` and
`test_config.yaml` files committed to this repo **can no longer be decrypted**. To redeploy
you must create a new KMS key and re-create those files from scratch with fresh credentials.
Everything else (application code, Dockerfiles, CloudFormation templates, Makefile) is intact
and reusable.

## What was removed vs. kept (teardown of June 2026)

| Resource | Status | Notes |
|---|---|---|
| CloudFormation stack `bandits-notification-v3` | **Deleted** | Lambda, EventBridge rule, SQS DLQ, exec role, log group |
| CloudFormation stack `github-actions-role` | **Deleted** | CI/CD deploy role `GitHubActions-BanditsNotification` |
| ECR repo `development/bandits-notification` | **Deleted** | Images rebuilt on next deploy |
| S3 bucket `bandits-notification-prod-data` | **Deleted** | Was empty |
| KMS key `BanditsNotifierKMSKey` (`72de9be2-…`) | **Deleted** | Old `secrets.yaml`/`test_config.yaml` are now permanently undecryptable |
| **S3 bucket `banditsnotifier-storage`** | **KEPT** | Schedule archive/history preserved (`bandits12u/`, `bandits14u/`) |
| IAM OIDC provider `token.actions.githubusercontent.com` | **KEPT** | Account-shared; not specific to this project |

Account: `028036396420` · Region: `us-east-1` · SSO profile: `developmentadmin`

---

## Step 1 — Create a new KMS key (reuse the old alias)

Re-using the alias `BanditsNotifierKMSKey` means nearly every reference in the repo keeps
working unchanged. Only one hardcoded raw key ID needs updating (Step 4).

```bash
PROFILE=developmentadmin
REGION=us-east-1

# Create the key
KEY_ID=$(aws kms create-key \
  --profile $PROFILE --region $REGION \
  --description "SOPS encryption key for Bandits Notification secrets" \
  --query 'KeyMetadata.KeyId' --output text)
echo "New KMS key ID: $KEY_ID"

# Attach the same alias the repo already references
aws kms create-alias \
  --profile $PROFILE --region $REGION \
  --alias-name alias/BanditsNotifierKMSKey \
  --target-key-id "$KEY_ID"
```

The default key policy grants the account root full access and lets IAM control usage, so your
`developmentadmin` identity can immediately encrypt/decrypt with it.

---

## Step 2 — Recreate `secrets.yaml` (plaintext)

Create a plaintext `secrets.yaml` in the repo root with the structure below. Replace every
`<...>` with a real value. (`make encrypt` in Step 3 will encrypt it in place.)

```yaml
aws:
    access_key_id: ""          # leave empty — Lambda uses its IAM execution role
    secret_access_key: ""      # leave empty
    region: us-east-1
    s3_bucket: banditsnotifier-storage
app:
    display_timezone: America/New_York
    urls:
        - url: <12U schedule page URL>           # Brookline Bandits 12U schedule page
          twitter:
            consumer_key: <12U app consumer key>
            consumer_secret: <12U app consumer secret>
            access_token: <12U access token>
            access_token_secret: <12U access token secret>
            user_handle: <12U account @handle, without the @>
        - url: <14U schedule page URL>           # Brookline Bandits 14U schedule page
          twitter:
            consumer_key: <14U app consumer key>
            consumer_secret: <14U app consumer secret>
            access_token: <14U access token>
            access_token_secret: <14U access token secret>
            user_handle: <14U account @handle, without the @>
```

### Where each value comes from

- **`s3_bucket`** — keep `banditsnotifier-storage`; the bucket and its archive were preserved,
  so change detection picks up right where it left off. The archive lives under `bandits12u/`
  and `bandits14u/` prefixes in that bucket.
- **`url`** — the public Brookline Bandits 12U and 14U schedule pages being monitored.
- **Twitter/X credentials** — regenerate from the X Developer Portal
  (<https://developer.twitter.com/en/portal/dashboard>) for each posting account:
  - `consumer_key` / `consumer_secret` = the App's **API Key and Secret**.
  - `access_token` / `access_token_secret` = **Access Token and Secret** generated for the
    account, with **Read and Write** permissions (required to post tweets + upload media).
  - `user_handle` = the account's @handle (no leading `@`).
  - Each schedule (12U, 14U) posts from its own account, so generate a separate token set per
    `urls` entry.

> The application reads this config via `internal/config/`. Field names above match the structs
> exactly — keep the nesting and key names as shown.

---

## Step 3 — Encrypt `secrets.yaml` with SOPS

The `Makefile` already points `KMS_KEY_ARN` at `alias/BanditsNotifierKMSKey`, so no edit is
needed if you reused the alias in Step 1:

```bash
make encrypt
```

This runs `sops -e --kms <KMS_KEY_ARN> -i secrets.yaml`, encrypting values in place. Verify:

```bash
make decrypt   # should print the plaintext back; re-run `make encrypt` afterward
git diff secrets.yaml   # confirm the sops: metadata now references the new key
```

(If you also run integration tests, recreate and encrypt `test_config.yaml` the same way using
`make encrypt-test` — see `test_config.yaml.example` for its structure.)

---

## Step 4 — Fix the one hardcoded raw key ID

`infrastructure/lambda-stack.yaml` references the **old** key's raw UUID in the Lambda KMS
policy. The alias is fine everywhere else, but this line must be updated to the new key's UUID
(the `$KEY_ID` printed in Step 1):

- File: `infrastructure/lambda-stack.yaml`
- Around line 65, in `KMSAccessPolicy`:
  ```yaml
  Resource:
    - !Ref KMSKeyArn
    - arn:aws:kms:us-east-1:028036396420:key/<OLD_UUID>   # <-- replace with new $KEY_ID
  ```

Replace `<OLD_UUID>` with the new key ID. (Alternatively, delete that second line entirely and
rely on `!Ref KMSKeyArn`, since the alias already resolves to the new key.)

No other code references the raw UUID. References that use the **alias** —
`Makefile`, `infrastructure/deploy.sh`, `infrastructure/lambda-stack.yaml` (the `KMSKeyArn`
default), and `infrastructure/github-actions-role.yaml` — all keep working as-is.

---

## Step 5 — Redeploy the infrastructure

`make lambda-deploy` runs `infrastructure/deploy.sh`, which **recreates the ECR repo**, builds
and pushes the Lambda image, and deploys the `bandits-notification-v3` CloudFormation stack
(Lambda + EventBridge schedule + SQS DLQ + exec role + log group).

```bash
# Make sure you're logged in to SSO
aws sso login --profile developmentadmin

# Build, push, and deploy everything
make lambda-deploy
```

`deploy.sh` already hardcodes `STACK_NAME=bandits-notification-v3`,
`ECR_REPOSITORY=development/bandits-notification`, `S3BucketName=banditsnotifier-storage`, and
the KMS alias — so a clean deploy reproduces the previous setup. The EventBridge rule is created
**ENABLED** at `rate(60 minutes)`, so notifications resume automatically.

### (Optional) Re-enable CI/CD auto-deploy

Only needed if you want pushes to `main` to auto-deploy again. Two parts:

1. **Recreate the GitHub Actions role.** The OIDC provider still exists, so just redeploy the
   role stack:
   ```bash
   aws cloudformation deploy \
     --profile developmentadmin --region us-east-1 \
     --template-file infrastructure/github-actions-role.yaml \
     --stack-name github-actions-role \
     --capabilities CAPABILITY_NAMED_IAM
   ```
2. **Re-enable the workflow triggers.** During shutdown, `.github/workflows/deploy.yml` was set
   to manual-only (`workflow_dispatch`) to avoid failed runs against torn-down infra. Uncomment
   the `push:` / `pull_request:` triggers at the top of that file to restore auto-deploy.

---

## Step 6 — Verify

```bash
# Stack is up
aws cloudformation describe-stacks --profile developmentadmin --region us-east-1 \
  --stack-name bandits-notification-v3 --query 'Stacks[0].StackStatus'

# Schedule rule is enabled
aws events list-rules --profile developmentadmin --region us-east-1 \
  --query "Rules[?Name=='bandits-notification-v3-schedule'].State"

# Invoke once manually and watch logs
make lambda-invoke
aws logs tail /aws/lambda/bandits-notification-v3-bandits-notification \
  --profile developmentadmin --region us-east-1 --follow
```

A healthy run logs S3 access success and a completed schedule check, writes new archives under
`banditsnotifier-storage`, and (if a change is detected) posts a tweet with a screenshot.

---

## Quick checklist

- [ ] Create new KMS key + alias `BanditsNotifierKMSKey` (Step 1)
- [ ] Recreate plaintext `secrets.yaml` with fresh Twitter tokens (Step 2)
- [ ] `make encrypt` (Step 3)
- [ ] Update raw key UUID in `infrastructure/lambda-stack.yaml` (Step 4)
- [ ] `make lambda-deploy` (Step 5)
- [ ] (Optional) redeploy `github-actions-role` stack + re-enable `deploy.yml` triggers
- [ ] Verify stack, schedule, and a manual invocation (Step 6)
