# coding=utf-8

import json
from flask import request, abort, g, jsonify, make_response
from flask_restful import Resource
from app.api.authentication import auth

from ..common.schema import UserSchema, FeedSchema
from marshmallow import ValidationError, pprint

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
        feed, errors = FeedSchema().load(request.json)
        res = FeedSchema().dumps(feed)
        pprint(res.data, indent=2)
        # return json.dumps(res.data, indent=2)
        response = make_response()
        response.mimetype = 'application/json'
        response.status_code = 200
        response.data = json.dumps(res.data, indent=4)
        return response
