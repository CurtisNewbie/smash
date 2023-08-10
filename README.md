# smash

Lets smash some endpoints

Specify where the instruction file is:

```sh
go build -o smasher
./smasher instruction-file=demo.yml
```

By default, the response body is logged in DEBUG level, enable it as follows:

```
./smasher instruction-file=demo.yml logging.level=INFO
```

Specify the rules in the instruction file:

```yaml
instructions:
  - url: "http://localhost:8080/ping" # instruction with cron that runs periodically
    method: PUT
    parallism: 100
    cron: "*/1 * * * * ?"
    headers:
      - "Content-Type": "application/json"
    payload: '{ "purpose": "get wrecked my boi!!!" }'

  - url: "http://localhost:8080/pong" # instruction without cron that only run once
    method: GET
    parallism: 100
    headers:
      - "Content-Type": "application/json"
```