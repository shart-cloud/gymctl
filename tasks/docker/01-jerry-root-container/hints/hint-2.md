## Hint 2: Create a Non-Root User

In Alpine, you can add a user like this:

```dockerfile
RUN adduser -D -u 1000 appuser
```

Then switch to it:

```dockerfile
USER appuser
```
