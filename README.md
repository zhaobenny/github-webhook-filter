# github-webhook-filter

Small Go app that filters out push events for specific branches from a GitHub webhook. (A solution to this [issue](https://stackoverflow.com/questions/46140233/github-webhooks-triggered-globally-instead-of-per-branch))

All other events will be forwarded like normal.

## Usage
See [docker-compose.yml](./docker-compose.yml)
