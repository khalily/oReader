# coding=utf-8

from flask_restful import Resource, marshal_with, fields
from flask.ext.wtf import Form
from wtforms import StringField, PasswordField
from wtforms.validators import Email, Regexp, ValidationError, EqualTo, Length, DataRequired
from app import db
from app.models import User


class RegisterForm(Form):
    email = StringField('Email', validators=[DataRequired(), Length(1, 64), Email()])
    password = PasswordField('Password', validators=[DataRequired(), Length(1, 16),
                             EqualTo('password2', message='password must match')])
    password2 = PasswordField('Confirm Password', validators=[DataRequired()])

    def validate_email(self, email):
        if User.query.filter_by(email=email).first():
            raise ValidationError('Email Already registered')

user_fields = {
    'email': fields.String
}


class User(Resource):
    @marshal_with(user_fields)
    def post(self):
        register_form = RegisterForm()
        if register_form.validate_on_submit():
            user = User(email=register_form.email.data)
            user.password = register_form.password.data
            db.session.add(user)
            return user

