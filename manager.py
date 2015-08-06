from app import create_app


app = create_app('devlopment')

if __name__ == '__main__':
    app.run()