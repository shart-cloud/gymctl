Look for fields that should come from the shared base and *not* differ per overlay:
- `spec.template.metadata.labels`
- container image tag

Keep `APP_MODE=prod` as the prod-specific override.
