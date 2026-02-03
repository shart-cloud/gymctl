# Hint 2: Create the ConfigMap

You need to create a ConfigMap named `app-config` with these keys:
- DATABASE_HOST
- DATABASE_PORT
- APP_MODE

Use the `kubectl create configmap` command with `--from-literal` flags.