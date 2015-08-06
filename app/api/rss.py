from . import api

from flask import request

@api.route('/add-subscription', methods=['POST'])
def add_subscription():
    pass