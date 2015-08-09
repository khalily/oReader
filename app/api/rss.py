from . import api
from .. import db
from ..models import Feed, Item
from flask import request, jsonify, abort, current_app

from urllib2 import urlopen
from ..feedparser import FeedParser


@api.route('/')
def test():
    return jsonify({'result': 'test'})

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
