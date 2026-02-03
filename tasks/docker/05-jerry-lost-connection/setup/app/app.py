#!/usr/bin/env python3
import os
import redis
from flask import Flask, jsonify

app = Flask(__name__)

# Get Redis connection from environment
REDIS_HOST = os.environ.get('REDIS_HOST', 'localhost')
REDIS_PORT = int(os.environ.get('REDIS_PORT', 6379))

def get_redis():
    return redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)

@app.route('/')
def home():
    return 'Hello from Jerry!'

@app.route('/health')
def health():
    return jsonify({'status': 'healthy'})

@app.route('/test-redis')
def test_redis():
    try:
        r = get_redis()
        r.set('test_key', 'hello')
        value = r.get('test_key')
        if value == 'hello':
            return jsonify({'status': 'success', 'message': 'Redis connection works!'})
        else:
            return jsonify({'status': 'error', 'message': 'Unexpected value'}), 500
    except redis.ConnectionError as e:
        return jsonify({'status': 'error', 'message': f'Cannot connect to Redis at {REDIS_HOST}:{REDIS_PORT}'}), 500
    except Exception as e:
        return jsonify({'status': 'error', 'message': str(e)}), 500

if __name__ == '__main__':
    print(f'Connecting to Redis at {REDIS_HOST}:{REDIS_PORT}')
    app.run(host='0.0.0.0', port=8080)
