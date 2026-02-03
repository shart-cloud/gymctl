# Hint 2: The selector is wrong

The Service is looking for pods with:
- `app: jerry-app`
- `environment: production`

But the pods have:
- `app: jerry-web`
- `version: v1`
- `team: jerry`

Update the Service selector to match the pod labels.