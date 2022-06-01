# mappend: append a single-arch image to a multi-arch image

`mappend` is a tool to create a multi-arch image from a signle-arch image

## Motivation

https://github.com/oras-project/oras/issues/237

## Usage

```
$ oras push localhost:5000/blob:amd64 blob-amd64.tar
$ oras push localhost:5000/blob:arm64 blob-arm64.tar
$ crane pull --format oci localhost:5000/blob:amd64 build/blob-amd64
$ crane pull --format oci localhost:5000/blob:arm64 build/blob-arm64
$ mappend build/blob build/blob-arm64 linux/arm64
$ mappend build/blob build/blob-amd64 linux/amd64
$ skopeo inspect --raw oci:./build/blob | jq .
{
  "schemaVersion": 2,
  "manifests": [
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "size": 420,
      "digest": "sha256:51d28b46526e08eb0cd895e11b38c4ba7474a075f89b4caa85081ef38bb77f48",
      "platform": {
        "architecture": "arm64",
        "os": "linux"
      }
    },
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "size": 420,
      "digest": "sha256:14bbf77921ab39a6e3284157a22ef7dc909e59047bb0cbe25fac979dc0bb9e43",
      "platform": {
        "architecture": "amd64",
        "os": "linux"
      }
    }
  ]
}
```

Or you can do the same without a registy by using the `oras-go` [example](https://github.com/oras-project/oras-go/tree/main/examples/advanced).

```
$ git clone https://github.com/oras-project/oras-go.git
$ cd oras-go/examples/advanced && go build
$ ./advanced copy blob:amd64 --from files blob-amd64.tar --to oci:build/blob-amd64
$ ./advanced copy blob:arm64 --from files blob-arm64.tar --to oci:build/blob-arm64
$ mappend build/blob build/blob-arm64 linux/arm64
$ mappend build/blob build/blob-amd64 linux/amd64
$ skopeo inspect --raw oci:./build/blob | jq .
{
  "schemaVersion": 2,
  "manifests": [
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "size": 420,
      "digest": "sha256:51d28b46526e08eb0cd895e11b38c4ba7474a075f89b4caa85081ef38bb77f48",
      "platform": {
        "architecture": "arm64",
        "os": "linux"
      }
    },
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "size": 420,
      "digest": "sha256:14bbf77921ab39a6e3284157a22ef7dc909e59047bb0cbe25fac979dc0bb9e43",
      "platform": {
        "architecture": "amd64",
        "os": "linux"
      }
    }
  ]
}
```

License
-
mappend is licensed under the Apache License, Version 2.0. See
[LICENSE](LICENSE) for the full license text.
