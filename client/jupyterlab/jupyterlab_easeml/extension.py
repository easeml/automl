import tornado.ioloop
import tornado.web
import os

class Handler(tornado.web.StaticFileHandler):
    def parse_url_path(self, url_path):
        if not url_path or url_path.endswith('/'):
            url_path = url_path + 'index.html'
        return url_path

def mkapp(prefix='',servepath=''):
    if not servepath:
        servepath=os.getcwd()

    if prefix:
        path = '/' + prefix + '/(.*)'
    else:
        path = '/(.*)'

    application = tornado.web.Application([
        (path, Handler, {'path': servepath}),
    ], debug=True)

    return application

def start_server(prefix='', port=8000):
    my_path = os.path.abspath(os.path.dirname(__file__))
    path=os.path.join(my_path,"web")
    app = mkapp(prefix,path)
    app.listen(port)

def load_jupyter_server_extension(nb_server_app):
    """
    Called when the extension is loaded.

    Args:
        nb_server_app (NotebookWebApplication): handle to the Notebook webserver instance.
    """
    PORT=8081

    nb_server_app.log.info('@Starting server easemlui service on port {}'.format(PORT))
    start_server(prefix='', port=PORT)
    nb_server_app.log.info('#Server for easemlui service started on port {}'.format(PORT))