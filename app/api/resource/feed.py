# coding=utf-8

import json
from flask import request, abort, g, jsonify, make_response
from flask_restful import Resource
from app.api.authentication import auth

from ..common.schema import UserSchema, FeedSchema
from marshmallow import ValidationError

@UserSchema.error_handler
def handle_errors(schema, errors, obj):
    raise ValidationError(errors)


class FeedList(Resource):
    @auth.login_required
    def get(self):
        result, errors = UserSchema().dump(g.current_user)
        if errors:
            abort(500)
        response = make_response()
        response.mimetype = 'application/json'
        response.status_code = 200
        response.data = json.dumps(result['feeds'], indent=2)
        return response

    @auth.login_required
    def post(self):
        feed = FeedSchema().validate(request.json)
        res = FeedSchema().dumps(feed)
        return jsonify(res.data)
