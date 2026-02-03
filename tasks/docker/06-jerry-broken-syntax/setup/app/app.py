#!/usr/bin/env python3

from flask import Flask, jsonify
import os

app = Flask(__name__)

@app.route('/')
def home():
    return jsonify({
        "status": "healthy",
        "message": "Jerry's app is running!",
        "debug": os.environ.get('DEBUG_MODE', 'false')
    })

@app.route('/health')
def health():
    return jsonify({"status": "ok"}), 200

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080)