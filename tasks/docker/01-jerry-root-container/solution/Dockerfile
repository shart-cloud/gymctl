FROM alpine:3.19

WORKDIR /app
COPY app/ /app/
RUN chmod +x /app/start.sh \
    && adduser -D -u 1000 appuser \
    && chown -R appuser:appuser /app

USER appuser

EXPOSE 8080
CMD ["/app/start.sh"]
