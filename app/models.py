#coding: utf-8

from app import db, login_manager
from flask import current_app
from flask.ext.login import UserMixin
from werkzeug.security import generate_password_hash, check_password_hash
from itsdangerous import TimedJSONWebSignatureSerializer as Serializer

class User(db.Model, UserMixin):
    __tablename__ = 'users'
    id = db.Column(db.Integer, primary_key=True)
    email = db.Column(db.String, nullable=False)
    password_hash = db.Column(db.String)

    def __init__(self, **kwargs):
        super(User, self).__init__(**kwargs)

    @property
    def password(self):
        raise AttributeError('password is not readable attribute')

    @password.setter
    def password(self, password):
        self.password_hash = generate_password_hash(password)

    def verify_password(self, password):
        return check_password_hash(self.password_hash, password)

    def generate_auth_token(self):
        s = Serializer(current_app.config['SECRET_KEY'], expires_in=3600)
        return s.dumps({'id': self.id})

    @staticmethod
    def verify_auth_token(token):
        s = Serializer(current_app.config['SECRET_KEY'])
        try:
            data = s.loads(token)
        except:
            return None
        return User.query.get(data['id'])


class Feed(db.Model):
    __tablename__ = 'feeds'
    id = db.Column(db.Integer, primary_key=True)
    title = db.Column(db.String)
    link = db.Column(db.String)
    description = db.Column(db.String)
    last_build_date = db.Column(db.String)
    img = db.Column(db.String)

    user_id = db.Column(db.Integer, db.ForeignKey('users.id'))
    user = db.relationship('User', backref=db.backref('feeds', order_by=id, lazy='dynamic'))

    def __repr__(self):
        return '<Feed>{id}{link}'.format(id=self.id, link=self.link)

    def to_json(self):
        return {
            'title': self.title,
            'link': self.link,
            'description': self.description,
            'last_build_date': self.last_build_date,
        }


class Item(db.Model):
    __tablename__ = 'items'
    id = db.Column(db.Integer, primary_key=True)
    title = db.Column(db.String)
    link = db.Column(db.String)
    description = db.Column(db.String)
    pub_date = db.Column(db.String)
    creator = db.Column(db.String)
    content = db.Column(db.String)
    updated = db.Column(db.String)
    feed_title = db.Column(db.String)
    star = db.Column(db.Boolean)

    feed_id = db.Column(db.Integer, db.ForeignKey('feeds.id'))
    feed = db.relationship('Feed', backref=db.backref('items', order_by=id, lazy='dynamic'))

    def to_json(self):
        return {
            'title': self.title,
            'link': self.link,
            'description': self.description,
            'pub_date': self.pub_date,
            'creator': self.creator,
            'content': self.content,
            'updated': self.updated,
        }


@login_manager.user_loader
def load_user(id):
    return User.query.get(id)
