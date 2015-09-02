from marshmallow import Schema, fields
from app.models import Feed, Item
from app import db
from flask import g, abort, Markup
from urllib2 import urlopen
from marshmallow import ValidationError
from app.api.errors import URLOpenError
from app.feedparser import FeedParser


class ItemSchema(Schema):
    content = fields.Method('safeContent')
    class Meta:
        fields = ('title', 'link', 'description', 'content', 'pub_date', 'creator', 'updated', 'feed_title')

    def safeContent(self, obj):
        return obj.content


class FeedSchema(Schema):
    items = fields.Nested(ItemSchema, many=True)
    url = fields.Url()
    class Meta:
        additional = ('title', 'link', 'description', 'last_build_date', 'img')

    def make_object(self, data):
        try:
            u = urlopen(data['url'])
        except:
            raise URLOpenError("Can't access Url")

        parser = FeedParser()
        if not parser.parse(u):
            abort(400)
        feed = Feed(**parser.feed)
        feed.user_id = g.current_user.id
        for item in parser.items:
            new_item = Item(**item)
            new_item.feed_title = feed.title
            feed.items.append(new_item)
        g.current_user.feeds.append(feed)
        db.session.add(g.current_user)
        db.session.commit()
        return feed


class UserSchema(Schema):
    feeds = fields.Nested(FeedSchema, many=True)
    email = fields.Email()
