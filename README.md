# smash

Lets smash some endpoints

Specify where the instruction file is:

```sh
go build -o smasher
./smasher -file demo.yml
```

By default, the response body is logged in DEBUG level, adjust logging level as below:

```
./smasher -file demo.yml -debug
```

Specify the rules in the instruction file:

```yaml
instructions:
  - url: "http://localhost:8080/ping" # instruction with cron that runs periodically
    method: PUT
    parallelism: 100
    cron: "*/1 * * * * ?"
    headers:
      - "Content-Type": "application/json"
    payload: '{ "purpose": "get wrecked my boi!!!" }'

  - url: "http://localhost:8080/pong" # instruction without cron that only run once
    method: GET
    parallelism: 100
    headers:
      - "Content-Type": "application/json"

  - parallelism: 100 # instruction that can be extracted from curl command
    curl: |
      curl 'http://localhost:8080/bang' \
      -H 'Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7' \
      -H 'Accept-Language: zh-CN,zh;q=0.9,en;q=0.8' \
      -H 'Cache-Control: no-cache' \
      -H 'Connection: keep-alive' \
      -H 'Cookie: Idea-f48890cc=c84b9839-f790-4bde-8194-62f9fd1c7013; Idea-f488948d=e4ed42e6-a875-414f-90ab-4de282d52ced; grafana_session=888ff110877475132284341b9779ffde; grafana_session_expiry=1691335938' \
      -H 'Pragma: no-cache' \
      -H 'Sec-Fetch-Dest: document' \
      -H 'Sec-Fetch-Mode: navigate' \
      -H 'Sec-Fetch-Site: none' \
      -H 'Sec-Fetch-User: ?1' \
      -H 'Upgrade-Insecure-Requests: 1' \
      -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36' \
      -H 'sec-ch-ua: "Not/A)Brand";v="99", "Google Chrome";v="115", "Chromium";v="115"' \
      -H 'sec-ch-ua-mobile: ?0' \
      -H 'sec-ch-ua-platform: "macOS"' \
      --compressed
```
