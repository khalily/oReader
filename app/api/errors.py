from . import api
from flask import jsonify


@api.errorhandler(400)
def bad_request(e):
    print 'bad_request', e
    return jsonify({'error': 'bad request'}), 400