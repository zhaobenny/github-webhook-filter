# 🌐 github-webhook-filter
Compact Go application (5 MB) designed to address a common GitHub webhook issue by filtering out push events for specific branches. 

(Solving the problem highlighted in this [Stack Overflow question](https://stackoverflow.com/questions/46140233/github-webhooks-triggered-globally-instead-of-per-branch))

All other events will seamlessly continue to be forwarded as usual.

## 🛠️ Usage

Copy the sample docker-compose.yml file for a swift setup.

Set the app's endpoint, `http://<HOST>:8080/forward`, in the `Payload URL` field within GitHub's `Add webhook` screen.

For the final webhooks destinations, add the URLs to the `WEBHOOKS` env within the Docker Compose file.

(Using the Github webhook `secret` option is also recommended!)

## License
[MIT](https://github.com/zhaobenny/github-webhook-filter/blob/main/LICENSE)
