# selektor

Template and execute SQL queries with JSON output.

## Configuring selektor

By default selektor's config is located in $HOME/.config/selektor/selektor.toml

selektor config contains of 2 entities: envs and selectors

example of selektor's config can be found in example directory

### env

It contains info needed to connect to database

Let's take a look at this env config

```toml
[env.mypg]
endpoint   = "localhost:5432"
type       = "postgres"
database   = "mydb"
user       = "myuser"
password   = "mypassword"
is_default = true
```

This env looks pretty self explanatory, there is only one peculiar flag: `is_default`.
This flag defines which env to use by default if no env is specified via `-e` common selektor flag.
Please note that there is only one env allowed to have `is_default` set to true.

It is also possible to include env config from another file (as shown in example config). If include option is set the env config will be read from given file.

```toml
[env.mypostgres]
include = "envs/mypostgres.toml"

[env.myclickhouse]
include = "/full/path/to/env/myclickhouse.toml"

[env.incorrectusage]
endpoint   = "localhost:5432"
type       = "postgres"
database   = "mydb"
user       = "myuser"
password   = "mypassword"
include    = "/full/path/to/env/myclickhouse.toml"
```

Please note that include option does not merge configs, so `incorrectusage` env in the example will override every option with those specified in `/full/path/to/env/myclickhouse.toml` file.

### selector

It contains info about queries and flags needed to template them.

Let's take a look at this selector config

```toml
[selector.user]
description = "get user by it's email or id"
query="""
    SELECT
        *
    FROM users
    WHERE
        1 = 1
        {{ .__defined_email }} AND email = '{{ .email }}'
        {{ .__defined_id }}    AND id = '{{ .id }}'
        {{ .__set_limit }}     LIMIT {{ .limit }}
"""

[selector.user.flag.id]
description = "specify user's id"
required = false

[selector.user.flag.email]
description = "specify user's email"
required = false

[selector.user.flag.limit]
description = "set limit for query"
required = false
default = "15"
```

The final query is templated with [go templates](https://golang.org/pkg/text/template/).

The template parameters can be set with flags that are defined for each selector.

Value of every declared flag can be passed to template using `{{ .flagname }}`

If the flag was declared and passed `{{ .__defined_flagname }}` will resolve in ""

If it was not passed, `{{ .__defined_flagname }}` will resolve in --, that can be used to comment out lines that filter your query with flags that were not passed

`{{ .__set_flagname }}` works in the same way as `{{ .__defined_flagname }}` except for default values. If flag was not set from command line but has a default (like `limit` in given example) `{{ .__set_flagname }}` will resolve in "".

To debug you query forming you can use `--showquery` common flag. It will only output your built query without executing it.

You can also use `--showtemplate` common flag to output resolved template parameters.

Here are outputs for user selector defined in example above

```bash
$ go run ./cmd/selektor/ -c example/selektor.toml user --showquery
    SELECT
        *
    FROM users
    WHERE
        1 = 1
        -- AND email = ''
        --    AND id = ''
             LIMIT 15

$ go run ./cmd/selektor/ -c example/selektor.toml user --email user@example.com --showquery
    SELECT
        *
    FROM users
    WHERE
        1 = 1
         AND email = 'user@example.com'
        --    AND id = ''
             LIMIT 15

$ go run ./cmd/selektor/ -c example/selektor.toml user --email user@example.com --showtemplate
{
    "__defined_email": "",
    "__defined_id": "--",
    "__defined_limit": "--",
    "__set_email": "",
    "__set_id": "--",
    "__set_limit": "",
    "__undefined_email": "--",
    "__undefined_id": "",
    "__undefined_limit": "",
    "__unset_email": "--",
    "__unset_id": "",
    "__unset_limit": "--",
    "email": "user@example.com",
    "id": "",
    "limit": "15"
}
```

There is also a way to define special flag types. Right now there is only one special flag type: `timerange`.

```toml
[selector.logs]
description = "get logs"
use_utc = true
query="""
    SELECT
        *
    FROM logs
    WHERE
        1 = 1
        {{ .__defined_ts }} AND timestamp > '{{ .tsFrom }}'
        {{ .__defined_ts }} AND timestamp < '{{ .tsTo }}'
        {{ .__set_limit }}  LIMIT {{ .limit }}
"""

[selector.logs.flag.ts]
description = "set time range for query"
required = true
type = "timerange"

[selector.logs.flag.limit]
description = "set limit for query"
required = false
default = "100"
```

Flags of timerange type are parsed in special way and form special template parameters `{{ .flagnameFrom }}` and `{{ .flagnameTo }}`

Here are some examples of timeranges

```
-1h/now
-1h (defaults to -1h/now)
15:33/now
15:33 (default to 15:33/now)
2020-08-08T12:00/2020-08-08T13:00
and many others that look the same
```

And and example of final query with timerange flag

```bash
$ go run ./cmd/selektor/ -c example/selektor.toml logs --ts -10d/now --showquery
    SELECT
        *
    FROM logs
    WHERE
        1 = 1
         AND timestamp > '2021-06-09 23:56:09'
         AND timestamp < '2021-06-19 23:56:09'
          LIMIT 100

$ go run ./cmd/selektor/ -c example/selektor.toml logs --ts -10d/now --limit 1000 --showtemplate
{
    "__defined_limit": "",
    "__defined_ts": "",
    "__set_limit": "",
    "__set_ts": "",
    "__undefined_limit": "--",
    "__undefined_ts": "--",
    "__unset_limit": "--",
    "__unset_ts": "--",
    "limit": "1000",
    "tsFrom": "2021-06-09 23:57:11",
    "tsTo": "2021-06-19 23:57:11"
}
```
