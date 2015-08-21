# coding=utf-8

import json
from flask import request, abort, g, jsonify, make_response
from flask_restful import Resource, marshal_with, fields
from app import db
from app.models import Feed as FeedModel, Item
from app.api.authentication import auth
from app.feedparser import FeedParser
from urllib2 import urlopen

item_fields = {
    'title': fields.String,
    'link': fields.String
}


feed_fields = {
    'title': fields.String,
    'link': fields.String,
    'description': fields.String,
    'last_build_data': fields.String,
    'items': fields.Nested(item_fields)
}

feeds_field = {
    'feeds': fields.Nested(feed_fields)
}


class FeedList(Resource):
    @auth.login_required
    def get(self):
        feeds = g.current_user.feeds
        response = make_response()
        response.mimetype = 'application/json'
        response.data = json.dumps([feed.to_json() for feed in feeds])
        response.status_code = 200
        return response

    @auth.login_required
    def post(self):
        url = request.json.get('url')
        try:
            u = urlopen(url)
        except Exception, e:
            abort(400)

        parser = FeedParser()
        if not parser.parse(u):
            abort(400)
        print parser.feed
        print parser.items
        feed = FeedModel(**parser.feed)
        feed.user_id = g.current_user.id
        for item in parser.items:
            feed.items.append(Item(**item))
        g.current_user.feeds.append(feed)
        db.session.add(g.current_user)

        return jsonify(feed.to_json())

