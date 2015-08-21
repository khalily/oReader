from flask import jsonify


def bad_request(msg):
    return jsonify({'error': 'bad request', 'message': msg}), 400

def forbidden(msg):
    return jsonify({'error': 'forbidden', 'message': msg}), 403

def unauthorized(msg):
    return jsonify({'error': 'unauthorized', 'message': msg}), 401

