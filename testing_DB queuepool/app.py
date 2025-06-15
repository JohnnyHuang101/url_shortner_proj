from flask import Flask, jsonify
from flask_sqlalchemy import SQLAlchemy
from sqlalchemy import event
import time

app = Flask(__name__)
app.config['SQLALCHEMY_DATABASE_URI'] = 'sqlite:///example.db'
app.config['SQLALCHEMY_TRACK_MODIFICATIONS'] = False

db = SQLAlchemy(app, engine_options={
    "echo": True,
    "pool_size": 3,         # max 3 concurrent connections
    "max_overflow": 1,      # only 1 additional overflow allowed
    "pool_timeout": 5       # timeout quickly for testing
})

# === Models ===

class User(db.Model):
    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(80), nullable=False)

class Occupation(db.Model):
    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(80), nullable=False)

class Department(db.Model):
    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(80), nullable=False)

class Task(db.Model):
    id = db.Column(db.Integer, primary_key=True)
    title = db.Column(db.String(80), nullable=False)
    user_id = db.Column(db.Integer, db.ForeignKey('user.id'))


# === Routes ===

@app.route('/create/<name>')
def create_user(name):
    user = User(name=name)
    db.session.add(user)
    db.session.commit()
    return f"User {name} created."

@app.route('/create_occupation/<name>')
def create_occupation(name):
    occupation = Occupation(name=name)
    db.session.add(occupation)
    db.session.commit()
    return f"Occupation {name} created."

@app.route('/create_department/<name>')
def create_department(name):
    dept = Department(name=name)
    db.session.add(dept)
    db.session.commit()
    return f"Department {name} created."

@app.route('/create_task/<title>/<int:user_id>')
def create_task(title, user_id):
    task = Task(title=title, user_id=user_id)
    db.session.add(task)
    db.session.commit()
    return f"Task '{title}' for user {user_id} created."


# Query multiple tables (but still one session)
@app.route('/query_all')
def query_all():
    users = User.query.all()
    occupations = Occupation.query.all()
    departments = Department.query.all()
    tasks = Task.query.all()

    time.sleep(10)  # Simulate long DB use
    return jsonify({
        'users': [u.name for u in users],
        'occupations': [o.name for o in occupations],
        'departments': [d.name for d in departments],
        'tasks': [t.title for t in tasks],
    })


# === DB Connection Pool Events ===

with app.app_context():
    @event.listens_for(db.engine, "checkout")
    def log_checkout(dbapi_conn, connection_record, connection_proxy):
        print("✅ [POOL] Connection checked OUT")

    @event.listens_for(db.engine, "checkin")
    def log_checkin(dbapi_conn, connection_record):
        print("🔁 [POOL] Connection checked IN")

    @event.listens_for(db.engine, "connect")
    def log_connect(dbapi_conn, connection_record):
        print("🔌 [POOL] New DBAPI connection created")


# === Start App ===
if __name__ == '__main__':
    with app.app_context():
        db.create_all()
    app.run(debug=True, threaded=True)
