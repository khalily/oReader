from . import api
from .. import db
from ..models import Feed, Item
from flask import request, jsonify, abort, current_app

from urllib2 import urlopen
from ..feedparser import FeedParser


@api.route('/')
def test():
    return jsonify({'result': 'test'})

@api.route('/test', methods=['POST'])
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
    db.session.add(Feed(**parser.feed))
    for item in parser.items:
        db.session.add(Item(**item))

    return jsonify({'result': 'ok'})


