## Hint 3: Permissions

If the app needs to write files, make sure the new user owns the app directory:

```dockerfile
RUN adduser -D -u 1000 appuser \
    && chown -R appuser:appuser /app
USER appuser
```
