# smash

Lets smash some endpoints

Specify where the instruction file is:

```sh
go run main.go instruction-file=demo.yml
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
    payload: '{ "purpose": "get wrecked my boi!!!" }'
```