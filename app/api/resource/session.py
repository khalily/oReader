# coding=utf-8
from flask import g, jsonify
from flask_restful import Resource
from app.api.authentication import auth
from app.api.errors import forbidden

class Session(Resource):
    @auth.login_required
    def get(self):
        if g.token_used:
            return forbidden('forbidden!!!')
        return jsonify({
            'token': g.current_user.generate_auth_token(),
            'profile': {
                'email': g.current_user.email
            }
        })

class TokenSession(Resource):
    @auth.login_required
    def get(self):
        if g.token_used:
            return jsonify({
                'profile': {
                    'email': g.current_user.email
                }
            })

