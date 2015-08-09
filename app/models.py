from app import db

class User(db.Model):
    __tablename__ = 'users'
    id = db.Column(db.Integer, primary_key=True)
    email = db.Column(db.String, nullable=False)


class Feed(db.Model):
    __tablename__ = 'feeds'
    id = db.Column(db.Integer, primary_key=True)
    title = db.Column(db.String)
    link = db.Column(db.String)
    description = db.Column(db.String)
    last_build_date = db.Column(db.String)

    user_id = db.Column(db.Integer, db.ForeignKey('users.id'))
    user = db.relationship('User', backref=db.backref('feeds', order_by=id, lazy='dynamic'))

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
