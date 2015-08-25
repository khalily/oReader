from flask import Blueprint
from flask_restful import Api

api_bp = Blueprint('api', __name__)

api = Api(api_bp)


from resource import FeedList, User, Session, TokenSession

api.add_resource(Session, '/get_token')
api.add_resource(TokenSession, '/login')
api.add_resource(FeedList, '/feeds/', '/feeds/<int:id>')

from . import errors, authentication