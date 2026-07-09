#!/bin/sh
# df-b entrypoint. The real secret arrives at runtime via env/mount, never baked
# into the image.
echo "df-b running (API_KEY present: ${API_KEY:+yes})"
exec sleep infinity
