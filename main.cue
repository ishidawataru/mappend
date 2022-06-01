package main

import (
	"list"
	"dagger.io/dagger"
	"dagger.io/dagger/core"

	"universe.dagger.io/go"
	"universe.dagger.io/docker"
	"universe.dagger.io/git"
)

dagger.#Plan & {
	client: filesystem: ".": {
		read: contents: dagger.#FS
	}

	actions: {
		build: go.#Build & {
			source:  client.filesystem.".".read.contents
			ldflags: "-extldflags \"-f no-PIC -static\""
			tags:    "osusergo netgo"
		}
		build_oras: #ORASBuild
		test:       docker.#Build & {
			steps: list.Concat([
				[
					docker.#Pull & {
						source: "homebrew/brew:latest"
					},
					docker.#Run & {
						command: {
							name: "brew"
							args: ["update"]
						}
					},
					docker.#Run & {
						command: {
							name: "brew"
							args: ["install", "skopeo", "jq"]
						}
					},
					docker.#Copy & {
						contents: build_oras.output
						dest:     "/usr/bin"
					},
					docker.#Copy & {
						contents: build.output
						dest:     "/usr/bin"
					},
				],
				list.Concat([
					for arch in ["amd64", "arm64"] {[
						// create a dummy blob for each arch
						#Shell & {script: "echo \(arch) > \(arch) && tar cvf \(arch).tar \(arch)"},
						// create a single-arch image in OCI format
						#Shell & {script: "advanced copy blob:\(arch) --from files \(arch).tar --to oci:build/blob-\(arch)"},
						// use mappend to create a multi-arch image
						#Shell & {script: "mappend build/blob build/blob-\(arch) linux/\(arch)"},
					]},
				]),
				[
					#Shell & {script: "/home/linuxbrew/.linuxbrew/bin/skopeo inspect --raw oci:./build/blob | jq '.manifests | length' > out.txt"},
					// the manifest length must be 2
					#Shell & {script: "[ $(cat out.txt) = 2 ]"},
				]])
		}
	}
}

#ORASBuild: {
	_pull: git.#Pull & {
		remote: "https://github.com/oras-project/oras-go.git"
		ref:    "main"
	}
	_build: docker.#Build & {
		steps: [
			docker.#Pull & {
				source: "golang:latest"
			},
			docker.#Copy & {
				contents: _pull.output
				dest:     "/app"
			},
			docker.#Run & {
				command: {
					name: "go"
					args: ["mod", "tidy"]
				}
				workdir: "/app"
			},
			docker.#Run & {
				command: {
					name: "go"
					args: ["build"]
				}
					workdir: "/app/examples/advanced"
				},
			]}
	_subdir: core.#Subdir & {
		input: _build.output.rootfs
		path:  "/app/examples/advanced/advanced"
	}
	output: _subdir.output
}

#Shell: {
	script: string
	docker.#Run & {
		command: {
			name: "sh"
			args: ["-c", script]
		}
		workdir: "/root"
		user:    "root"
	}
}
