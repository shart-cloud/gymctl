FROM python:3.11-alpine

WORKDIR /app

RUN pip install redis flask

COPY app.py .

EXPOSE 8080
CMD ["python", "app.py"]
