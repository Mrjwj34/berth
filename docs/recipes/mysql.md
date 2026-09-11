# Recipe: MySQL as a process

## When to use

The project needs a local MySQL (or MariaDB) per workspace. lane has no MySQL type — declare `mysqld` as a process, bind `--port` to `LANE_PORT_MYSQL`, and keep `--datadir` under `LANE_DATA_DIR`.

If MySQL already binds `3306`, declare `listen: 3306` and keep the command unchanged. If the binary already accepts `--port`, you may pass `$LANE_PORT_MYSQL` instead. Discover the host-published port with `lane ports --json`.

## `lane.yaml` fragment

```yaml
ports: [mysql]
env:
  DATABASE_URL: mysql://root@127.0.0.1:${LANE_PORT_MYSQL}/app
  MYSQL_UNIX_PORT: ${LANE_DATA_DIR}/mysql.sock
hooks:
  setup:
    - mkdir -p "$LANE_DATA_DIR/mysql"
    - test -d "$LANE_DATA_DIR/mysql/mysql" || mysqld --initialize-insecure --datadir="$LANE_DATA_DIR/mysql"
processes:
  mysql:
    command: mysqld --datadir="$LANE_DATA_DIR/mysql" --port="$LANE_PORT_MYSQL" --socket="$LANE_DATA_DIR/mysql.sock" --mysqlx=0 --bind-address=127.0.0.1
    readiness_probe:
      exec:
        command: mysqladmin ping --protocol=TCP --host=127.0.0.1 --port="$LANE_PORT_MYSQL" --silent
```

`--initialize-insecure` is for local Agent workspaces only. Do not print credentials. Create the app database in `hooks.setup` or with `lane run --` after the probe is healthy.
