from app import db

class User(db.Model):
    __tablename__ = 'users'
    id = db.Column(db.Integer, primary_key=True)
    email = db.Column(db.String(64), nullable=False)


class Feed(db.Model):
    __tablename__ = 'feeds'
    id = db.Column(db.Integer, primary_key=True)
    xml_name = db.Column(db.String(128))
    title = db.Column(db.String(64))
    link = db.Column(db.String(128))
    description = db.Column(db.String(256))
    pub_date = db.Column(db.String(64))
    last_build_date = db.Column(db.DateTime)

    user_id = db.ForeignKey('users.id')
    user = db.relationship('User', backref=db.backref('feeds', order_by=id))


class Item(db.Model):
    __tablename__ = 'items'
    id = db.Column(db.Integer, primary_key=True)
    title = db.Column(db.String(64))
    link = db.Column(db.String(128))
    description = db.Column(db.String(256))
    pub_date = db.Column(db.String(64))
    last_build_date = db.Column(db.DateTime)

    feed_id = db.Column(db.ForeignKey('feeds.id'))
    feed = db.relationship('Feed', backref=db.backref('items', order_by=id))
