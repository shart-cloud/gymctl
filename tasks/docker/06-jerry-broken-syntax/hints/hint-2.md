# Hint 2: Specific issues to fix

1. **Line 1**: `from` should be `FROM`
2. **Lines 9-10**: Missing backslash for line continuation in RUN command
3. **Line 14**: `env` should be `ENV`
4. **Lines 16-17**: You can only have one CMD instruction - remove the duplicate
5. **Line 28**: RUN commands can't modify files after USER is set