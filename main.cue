package main

import (
	"list"
	"dagger.io/dagger"

	"universe.dagger.io/go"
	"universe.dagger.io/docker"
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
		test: docker.#Build & {
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
						contents: build.output
						dest:     "/usr/bin"
					},
				],
				list.Concat([
					for arch in ["amd64", "arm64"] {[
						// create a dummy blob for each arch
						#Shell & {script: "echo \(arch) > \(arch) && tar cvf \(arch).tar \(arch)"},
						// use mappend to create a multi-arch image
						#Shell & {script: "mappend build/blob \(arch).tar linux/\(arch)"},
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
