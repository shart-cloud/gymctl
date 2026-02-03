FROM python:3.11-slim

WORKDIR /app

# Handle both old and new Python environments
RUN pip install --no-cache-dir --break-system-packages redis flask 2>/dev/null || \
    pip install --no-cache-dir redis flask

COPY app.py .

EXPOSE 8080
CMD ["python", "app.py"]
