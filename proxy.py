
from flask import Flask, request, Response
import requests
from flask_cors import CORS


app = Flask(__name__)
CORS(app)

@app.route('/proxy')
def proxy():
    url = request.args.get('url')
    if not url or not (url.startswith('http://') or url.startswith('https://')):
        return Response('Invalid or missing url parameter', status=400)
    try:
        resp = requests.get(url)
        return Response(resp.content, status=resp.status_code, content_type=resp.headers.get('Content-Type', 'application/octet-stream'))
    except Exception as e:
        return Response(f'Error: {e}', status=500)

if __name__ == '__main__':
    app.run(port=5001, debug=True)
