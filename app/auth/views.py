from . import auth
from forms import LoginForm
from flask import redirect, url_for, render_template
from flask.ext.login import login_user, logout_user
from ..models import User


@auth.route('/login', methods=['POST'])
def login():
    form = LoginForm()
    if form.validate_on_submit():
        user = User.query.filter_by(email=form.email.data).first()
        if user is not None:
            login_user(user)
            return redirect(url_for('main.index'))
    return render_template('index.html')
