from . import api
from .. import db
from ..models import Feed, Item
from flask import request, jsonify, abort, current_app, g
from authentication import auth
from errors import unauthorized

from urllib2 import urlopen
from ..feedparser import FeedParser


@api.before_request
@auth.login_required
def before_request():
    pass

@api.route('/get_token')
def get_token():
    if g.token_used:
        return unauthorized('Invalid credentials')
    return jsonify({ 'token': g.current_user.generate_auth_token() })

@api.route('/add-subscription', methods=['POST'])
def add_subscription():
    url = request.json.get('url')
    print url
    try:
        u = urlopen(url)
    except Exception, e:
        print e.message
        abort(400)

    parser = FeedParser()
    if not parser.parse(u):
        print 'parse false'
        abort(400)
    print parser.feed
    print parser.items
    print current_app.config['SQLALCHEMY_DATABASE_URI']
    feed = Feed(**parser.feed)
    for item in parser.items:
        feed.items.append(Item(**item))
    db.session.add(feed)

    return jsonify({'result': 'ok'})

@api.route('/get-story/')
def get_story():
    data = []
    for feed in Feed.query.all():
        print feed.to_json()
        print feed.items.all()
        data.append(
            {
                "feed": feed.to_json(),
                "stories": [ item.to_json() for item in feed.items.all()],
            }
        )
    return jsonify({'feeds': data})
