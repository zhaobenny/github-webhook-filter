# 🌐 github-webhook-filter
Compact Go application (5 MB) designed to address a common GitHub webhook issue by filtering out push events for specific branches. 

(Solving the problem highlighted in this [Stack Overflow question](https://stackoverflow.com/questions/46140233/github-webhooks-triggered-globally-instead-of-per-branch))

## 🛠️ Usage

Copy the sample docker-compose.yml file for a swift setup.

Set the app's endpoint, `http://<HOST>:8080/forward`, in the `Payload URL` field within GitHub's `Add webhook` screen.

For the final webhooks destinations, add the URLs to the `WEBHOOKS` env within the Docker Compose file.

By default, the forwarder accepts only pushes with commits to existing allowlisted branches. It skips force-pushes, branch creation and deletion, tag pushes, empty pushes, and other webhook events. Set `COMMIT_PUSH_ONLY=false` in the environment or Compose `.env` file to filter by branch alone.

Use `WHITELISTED_BRANCHES` for comma-separated exact names and `WHITELISTED_BRANCH_REGEX` for a Go regular expression. A match in either setting allows the branch. The regex sees the branch name without `refs/heads/`. Use anchors for a full-name match. An invalid regex stops startup with a configuration error.

This Compose `.env` example allows `main` and branches under `release/`. Set `WHITELISTED_BRANCHES=` to use only the regex.

```dotenv
WHITELISTED_BRANCHES=main
WHITELISTED_BRANCH_REGEX='^release/.+$'
```

(Using the Github webhook `secret` option is also recommended!)

## License
[MIT](https://github.com/zhaobenny/github-webhook-filter/blob/main/LICENSE)
