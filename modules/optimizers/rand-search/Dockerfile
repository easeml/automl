FROM python:3.7-alpine3.7

RUN adduser -D easeml
USER easeml

WORKDIR /home/easeml

COPY requirements.txt ./

RUN pip install --no-cache-dir -r requirements.txt

COPY . .

ENTRYPOINT [ "python" ]
