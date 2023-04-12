# tendermint-exporter

Prometheus exporter for Tendermint chains

This exporter was build with regards to:

- https://gitlab.com/ipiton/tendermint-exporter
- https://github.com/solarlabsteam/tendermint-exporter


Prior to (cosmos-exporte)[https://github.com/solarlabsteam/tendermint-exporter] focusing on chain information, **temdermint-exporter** tries to export a Tendermint framework Node information (Cosmos Chain is built upon).

## How can I set it up?

First of all, you need to download the latest release from [the releases page](https://github.com/solarlabsteam/tendermint-exporter/releases/). After that, you should unzip it and you are ready to go:

```sh
wget <the link from the releases page>
tar xvfz tendermint-exporter-*
./tendermint-exporter
```

That's not really interesting, what you probably want to do is to have it running in the background. For that, first of all, we have to copy the file to the system apps folder:

```sh
sudo cp ./tendermint-exporter /usr/bin
```

Then we need to create a systemd service for our app:

```sh
sudo nano /etc/systemd/system/tendermint-exporter.service
```

You can use this template (change the user to whatever user you want this to be executed from. It's advised to create a separate user for that instead of running it from root):

```
[Unit]
Description=Cosmos Exporter
After=network-online.target

[Service]
User=<username>
TimeoutStartSec=0
CPUWeight=95
IOWeight=95
ExecStart=tendermint-exporter
Restart=always
RestartSec=2
LimitNOFILE=800000
KillSignal=SIGTERM

[Install]
WantedBy=multi-user.target
```

Then we'll add this service to the autostart and run it:

```sh
sudo systemctl enable tendermint-exporter
sudo systemctl start tendermint-exporter
sudo systemctl status tendermint-exporter # validate it's running
```

If you need to, you can also see the logs of the process:

```sh
sudo journalctl -u tendermint-exporter -f --output cat
```

## How can I scrape data from it?

Here's the example of the Prometheus config you can use for scraping data:

```yaml
scrape-configs:
  # specific validator(s)
  - job_name:       'tendermint'
    scrape_interval: 15s
    metrics_path: /metrics
    static_configs:
      - targets:
        - <list of validators you want to monitor>
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_address
      - source_labels: [__param_address]
        target_label: instance
      - target_label: __address__
        replacement: <node hostname or IP>:9301
```

Then restart Prometheus and you're good to go!

The exporter provides the following metrics:

- `tendermint_latest_block_height` - the block latest height
- `tendermint_latest_block_time` - the block latest time
- `tendermint_latest_block_time_diff` - the block latest time difference from the time of scrapping
- `tendermint_peers` - number of connected peers

## How does it work?

It queries the full node via gRPC and returns it in the format Prometheus can consume.


## How can I configure it?

You can pass the artuments to the executable file to configure it. Here is the parameters list:

- `--listen-address` - the address with port the node would listen to. For example, you can use it to redefine port or to make the exporter accessible from the outside by listening on `127.0.0.1`. Defaults to `:9301` (so it's accessible from the outside on port 9301)
- `--node` - the gRPC node URL. Defaults to `localhost:9090`
- `--tendermint-rpc` - Tendermint RPC URL to query node stats (specifically `chain-id`). Defaults to `http://localhost:26657`
- `--log-devel` - logger level. Defaults to `info`. You can set it to `debug` to make it more verbose.
- `--limit` - pagination limit for gRPC requests. Defaults to 1000.
- `--json` - output logs as JSON. Useful if you don't read it on servers but instead use logging aggregation solutions such as ELK stack.

Additionally, you can pass a `--config` flag with a path to your config file (I use `.toml`, but anything supported by [viper](https://github.com/spf13/viper) should work).

## Which networks this is guaranteed to work?

In theory, it should work on a Cosmos-based blockchains with cosmos-sdk >= 0.40.0 (that's when they added gRPC and IBC support). If this doesn't work on some chains, please file and issue and let's see what's up.

## How can I contribute?

Bug reports and feature requests are always welcome! If you want to contribute, feel free to open issues or PRs.
