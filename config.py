import os

basedir = os.path.abspath(os.path.dirname(__file__))

class Config:
    SQLALCHEMY_COMMIT_ON_TEARDOWN = True
    SECRET_KEY = "123456"
    @staticmethod
    def init_app(app):
        pass

class DevlopmentConfig(Config):
    DEBUG = True
    SQLALCHEMY_DATABASE_URI = 'sqlite:///' + \
        os.path.join(basedir, 'dev-data.sqlite')

class TestingConfig(Config):
    DEBUG = True
    SQLALCHEMY_DATABASE_URI = 'sqlite:///' + \
        os.path.join(basedir, 'test-data.sqlite')

class ProductionConfig(Config):
    DEBUG = False
    SECRET_KEY = "abcdefg"
    SQLALCHEMY_DATABASE_URI = 'sqlite:///' + \
        os.path.join(basedir, 'product-data.sqlite')

config = {
    'production': ProductionConfig,
    'devlopment': DevlopmentConfig,
    'testing': TestingConfig,
}
