FROM python:3.7

WORKDIR /usr/src/app

COPY requirements.txt ./

RUN pip install --extra-index-url https://test.pypi.org/simple/ --no-cache-dir -r requirements.txt

COPY . .

ENTRYPOINT [ "python" ]
