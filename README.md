# kube-replay

This is the GraphQL backend for Kube-Replay that uses AWS DynamoDB as the persistence.

## DyanmoDB Setup
After obtaining the necessary AWS temporary credentials (i.e. `credentials` file), run
```text
go run setup.go <tableName>
```
where `tableName` will be the name of the DynamoDB table. It should be 1 table per Kubernetes cluster so therefore 
it's recommended that the `tableName` should just be the cluster name.

## Running GraphQL Server
```text
go run server.go
```

## Sample query

```graphql
query NODE_SNAPSHOT_NOW {
    nodeStatesAtTimestamp(timestamp: "2025-04-27T00:00:00Z") {
      timestamp
      nodes {
        id
        timestamp
        name
        roles
        providerID
        info {
          architecture
          containerRuntimeVersion
          kernelVersion
          kubeletVersion
          kubeProxyVersion
          osImage
          operatingSystem
          machineId
          systemUUID
          bootID
        }
        state {
          status
          capacity {
            cpu
            memory
            ephemeralStorage
            pods
          }
          allocatable {
            cpu
            memory
            ephemeralStorage
            pods
          }
          taints {
            key
            value
            effect
          }
        }
        pods {
          id
          timestamp
          name
          namespace
          status
          nodeID
          startedAt
          deletedAt
          finishedAt
          deletedBy
          qosClass
          containers {
            containerID
            name
            image
            imageID
            resources {
              requests {
                cpu
                memory
                ephemeralStorage
              }
              limits {
                cpu
                memory
                ephemeralStorage
              }
            }
            ready
            restartCount
            startedAt
            running
            state {
              exitCode
              startedAt
              finishedAt
              reason
            }
            lastState {
              exitCode
              startedAt
              finishedAt
              reason
            }
          }
        }
      }
    }
  }
```