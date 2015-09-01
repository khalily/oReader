from . import main
from flask import render_template, redirect, url_for

@main.route('/')
def index():
    return render_template('index.html')

@main.route('/favicon.ico')
def favicon():
    return redirect(url_for('static', filename='favicon.ico'))
