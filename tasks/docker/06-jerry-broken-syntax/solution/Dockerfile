FROM ubuntu:latest

RUN apt-get update && \
    apt-get install --break-system-packages python3 python3-pip curl

WORKDIR /app

COPY app/ /app

RUN pip3 install --break-system-packages flask==2.0.1 && \
    pip3 install --break-system-packages requests

RUN chown -R nobody:nobody /app

USER nobody

EXPOSE 8080

ENV FLASK_APP=app.py
ENV DEBUG_MODE=true

HEALTHCHECK --interval=30s --timeout=3s \
  CMD curl -f http://localhost:8080/ || exit 1

ENTRYPOINT ["python3"]
CMD ["app.py"]