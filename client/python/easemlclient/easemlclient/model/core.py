"""[summary]
"""
import requests

from requests.auth import HTTPBasicAuth
from typing import Optional
import json

API_PREFIX = "api/v1"
API_KEY_HEADER = "X-API-KEY"

class Connection:

    def __init__(
            self,
            host: str,
            user_id: Optional[str] = None,
            user_password: Optional[str] = None,
            api_key: Optional[str] = None) -> None:
        if (user_id is None or user_password is None) and api_key is None:
            raise ValueError("A connection instance must be initialized with either an API " + \
                             "key or a user id and password.")

        self.host = host.strip("/")
        self.user_id = user_id
        self.api_key = api_key

        if api_key is not None:
            def api_key_auth(request):
                request.headers[API_KEY_HEADER] = api_key
                return request
            self.auth = api_key_auth
        else:
            self.auth = HTTPBasicAuth(user_id, user_password)

    @property
    def url_base(self):
        return self.host + "/" + API_PREFIX

    def url(self, endpoint: str) -> str:
        return self.url_base + "/" + endpoint.lstrip("/")

    def login(self) -> 'Connection':
        # Login and update API KEY if needed
        url = self.url("users/login")
        resp = requests.get(url, auth=self.auth)
        resp.raise_for_status()
        api_key = resp.headers[API_KEY_HEADER]
        # Get current user if missing
        if not self.user_id:
            url = self.url("users/this")
            resp = requests.get(url, auth=self.auth)
            resp.raise_for_status()
            resp_dict = json.loads(resp.content)
            self.user_id = resp_dict['data']['id']

        return Connection(
            user_id=self.user_id,
            host=self.host,
            api_key=api_key)

    def logout(self):
        url = self.url("users/logout")
        resp = requests.get(url, auth=self.auth)
        resp.raise_for_status()
        self.api_key = None