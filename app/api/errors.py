from . import api
from flask import jsonify


def bad_request(msg):
    return jsonify({'error': 'bad request', 'message': msg}), 400

def forbidden(msg):
    return jsonify({'error': 'forbidden', 'message': msg}), 403

def unauthorized(msg):
    return jsonify({'error': 'unauthorized', 'message': msg}), 401

@api.errorhandler(405)
def method_not_allowed():
    return jsonify({'error': 'method not allowed', 'message': 'This method is not implementation'}), 405
