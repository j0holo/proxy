# Proxy - Link Tracker Project

This API proxy is part of the Link Tracker Project.

**This is a https only proxy and needs a TLS cert and key file.**

This is a fairly simple proxy that returns the answer of a server
in JSON format. You must authenticate with a API key set
in the `Authorization` header.

The following endpoints are available:

**POST /:**
The post request can look like this:

```
{
    "url": "http://example.com/",
    "header": {
        "Accept": ["*/*"],
        "User-Agent": ["ExampleProxy"]
    }
}
```

A valid response from the proxy will always be a status code 200.
If the `status` key is not an empty string something went wrong while fetching the page and most
keys will be empty.
A response can look like this:

```
{
    "url": "https://example.com/",
    "status_code": 200,
    "header": {
        "Accept-Ranges": [
            "bytes"
        ],
        "Cache-Control": [
            "max-age=604800"
        ],
        "Content-Type": [
            "text/html; charset=UTF-8"
        ],
        "Date": [
            "Sat, 01 Sep 2018 18:42:28 GMT"
        ],
        "Etag": [
            "\"1541025663\""
        ],
        "Expires": [
            "Sat, 08 Sep 2018 18:42:28 GMT"
        ],
        "Last-Modified": [
            "Fri, 09 Aug 2013 23:54:35 GMT"
        ],
        "Server": [
            "ECS (dca/24A0)"
        ],
        "Vary": [
            "Accept-Encoding"
        ],
        "X-Cache": [
            "HIT"
        ]
    },
    "body": "<!doctype html>\n<html>\n<head>\n    <title>Example Domain</title>\n\n    <meta charset=\"utf-8\" />\n    <meta http-equiv=\"Content-type\" content=\"text/html; charset=utf-8\" />\n    <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />\n    <style type=\"text/css\">\n    body {\n        background-color: #f0f0f2;\n        margin: 0;\n        padding: 0;\n        font-family: \"Open Sans\", \"Helvetica Neue\", Helvetica, Arial, sans-serif;\n        \n    }\n    div {\n        width: 600px;\n        margin: 5em auto;\n        padding: 50px;\n        background-color: #fff;\n        border-radius: 1em;\n    }\n    a:link, a:visited {\n        color: #38488f;\n        text-decoration: none;\n    }\n    @media (max-width: 700px) {\n        body {\n            background-color: #fff;\n        }\n        div {\n            width: auto;\n            margin: 0 auto;\n            border-radius: 0;\n            padding: 1em;\n        }\n    }\n    </style>    \n</head>\n\n<body>\n<div>\n    <h1>Example Domain</h1>\n    <p>This domain is established to be used for illustrative examples in documents. You may use this\n    domain in examples without prior coordination or asking for permission.</p>\n    <p><a href=\"http://www.iana.org/domains/example\">More information...</a></p>\n</div>\n</body>\n</html>\n",
    "status": ""
}
```

**GET /check:**
A simple GET check for HaProxy health checking.

**GET /stats:**
Not working