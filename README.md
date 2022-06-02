# mappend: create a multi-arch image from tarball

`mappend` is a tool to create a multi-arch image from tarball

## Motivation

https://github.com/oras-project/oras/issues/237

## Usage

```
$ mappend multi arm64.tar linux/arm64
$ mappend multi amd64.tar linux/amd64
$ skopeo inspect --raw oci:./multi | jq .
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
