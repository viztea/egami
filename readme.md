# egami

smol image sharing server written in Go.

## usage

`POST` multipart form data with your image file(s) to `/upload` endpoint. The form field name for the file must be `file`.

Example using `curl`:

```bash
curl -X POST -H 'Authorization: Bearer my-user-token' -F "file=@/path/to/your/image.jpg" http://localhost:3333/upload
```

The server will respond with a JSON object containing the URLs of the uploaded images.

```json
{
  "uploads": [
    {
      "id": "21d0534e",
      "filename": "21d0534e.jpg"
    }
  ]
}
```

## configuration

The server can be configured using the following environment variables:

- `EGAMI_SERVER_HOST`: The host address to bind the server to (default: `127.0.0.1`)
- `EGAMI_SERVER_PORT`: The port to bind the server to (default: `3333`)
- `EGAMI_DATA_DIRECTORY`: The directory to store uploaded images (default: `/var/lib/egami/data`)
- `EGAMI_USER_TOKEN`: The token required for authorization

---

[viztea](https://vzt.gay) &copy; 2025
