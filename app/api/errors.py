import json
from flask import jsonify, make_response
from marshmallow import ValidationError
from . import api_bp

class URLOpenError(Exception):
    pass

@api_bp.errorhandler(ValidationError)
def validate_error(error):
    return bad_request(error.message)

@api_bp.errorhandler(URLOpenError)
def urlopen_error(error):
    return bad_request(error.message)

def bad_request(msg):
    return jsonify({'error': 'bad request', 'message': msg}), 400

def forbidden(msg):
    return jsonify({'error': 'forbidden', 'message': msg}), 403

def unauthorized(msg):
    return jsonify({'error': 'unauthorized', 'message': msg}), 401

