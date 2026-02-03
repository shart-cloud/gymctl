# Hint 3: Complete solution approach

Fix these issues in order:

1. Change `from ubuntu:latest` to `FROM ubuntu:latest`
2. Fix the multi-line RUN command:
   ```dockerfile
   RUN pip3 install flask==2.0.1 && \
       pip3 install requests
   ```
3. Change `env DEBUG_MODE true` to `ENV DEBUG_MODE=true`
4. Remove duplicate CMD - keep only one (either line 16 or 22)
5. If using both ENTRYPOINT and CMD, CMD provides arguments to ENTRYPOINT
6. Move the `RUN chown` command before the `USER nobody` instruction
7. Install `curl` in the apt-get for the HEALTHCHECK to work