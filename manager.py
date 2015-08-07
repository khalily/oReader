from app import create_app, db
from app.models import Feed, Item, User

from flask.ext.script import Manager


app = create_app('devlopment')

manager = Manager(app)

@manager.shell
def make_context():
    return dict(db=db, Feed=Feed, Item=Item, User=User)

if __name__ == '__main__':
    manager.run()
