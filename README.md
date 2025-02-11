# Nexqloud Chain

## Deployment

- Clone the repository and switch to the upgrade branch
```
git clone https://github.com/nexqloud/nexqloud-chain.git
git switch upgrade
```

- Build the docker image
```
docker build -t nexqloud-chain .
```

- Set the environment variables
```shell
export MONIKER="node1" # The name of the node as it will appear in the network
export SEED_NODE_IP="" # The IP address of the seed node
export HOME = "/root" # The home directory of the container required to mount the volume
```

- Run the docker container
```shell
docker run -d --name nexqloud-chain -e MONIKER -e SEED_NODE_IP nexqloud-chain
```

- To have persistent data, you can mount a volume to the container which must map with the `~/.nxqd` directory
```shell
docker run -d --name nexqloud-chain -e MONIKER -e SEED_NODE_IP -v /path/to/local/directory:/root/.nxqd nexqloud-chain
```