#coding: utf-8

from flask.ext.wtf import Form
from wtforms.fields import StringField, SubmitField, PasswordField
from wtforms.validators import data_required


class LoginForm(Form):
    email = StringField("邮箱", validators=[data_required()])
    password = PasswordField("密码", validators=[data_required()])
    submit = SubmitField("登录")
