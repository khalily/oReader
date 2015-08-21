# coding=utf-8

from ..models import User
from flask import g, make_response
from errors import unauthorized
from flask.ext.httpauth import HTTPBasicAuth

auth = HTTPBasicAuth()


@auth.error_handler
def unauthorized():
    response = make_response()
    response.status_code = 401
    response.headers['WWW-Authenticate'] = 'xBasic realm="{0}"'.format('Authentication Required')
    print response
    return response

@auth.verify_password
def verify_password(email_or_token, password):
    if password == '':
        g.current_user = User.verify_auth_token(email_or_token)
        g.token_used = True
        return g.current_user != None
    user = User.query.filter_by(email=email_or_token).first()
    if not user:
        return False
    g.current_user = user
    g.token_used = False
    return user.verify_password(password)
