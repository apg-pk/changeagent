# ChangeAgent Apigee Factory Build Instructions

**Build and run Go application**

```
cd changeagent/
make

# Run standalone
mkdir data
./agent/agent -p 9000 -d ./data -logtostderr
# I0629 17:52:30.728034   27182 rocksdbstorage.go:69] Opened RocksDB file in ./data
# I0629 17:52:30.728930   27182 raftimpl.go:182] Node 7129430745073504080 starting
# I0629 17:52:30.730616   27182 raftimpl.go:199] Only one node. Starting in leader mode.
# I0629 17:52:30.732060   27182 mainloop.go:62] Node 7129430745073504080 entering leader mode
# I0629 17:52:30.733857   27182 main.go:80] Listening on port 9000

```

**Test Go application works**

```
# Post a new change
curl http://localhost:9000/changeagent/changes    \
  -H "Content-Type: application/json" \
  -d '{"data":{"Hello":"world"}}'
# {"_id":2}

# Retrieve all changes
curl http://localhost:9000/changeagent/changes
# [{"_id":2,"_ts":1467250861976242581,"data":{"Hello":"world"}}]
```

**Build docker image and run container**

```
docker build -t changeagent:latest . && docker images changeagent:latest

C=$(docker run -d -p 38080:8080 changeagent:latest)

# get container info
docker ps -f id=$C

# tail logs
docker logs -f $C
```

**Rebuild container and image (after writing some code)**

```
# get container ID if not known
C=$(docker ps -a -q -f ancestor=changeagent:latest)
docker stop $C && docker rm $C && docker rmi changeagent:latest

docker build -t changeagent:latest . && C=$(docker run -d -p 38080:8080 changeagent:latest)
```

**Test Go application works in docker container**

```
# open shell
docker exec -it $C bash

# if container not known, open shell for newest one
C=$(docker ps -a -q -f ancestor=changeagent:latest) && docker exec -it $C bash

  curl 0:8080/changeagent/changes
  # []
  
  curl 0:8080/changeagent/changes     \
  -H "Content-Type: application/json" \
  -d '{"data":{"Hello":"world"}}'
  # {"_id":2}
  
  curl 0:8080/changeagent/changes
  # [{"_id":2,"_ts":1467251552807965406,"data":{"Hello":"world"}}]
```

**Test Go application in docker from dev laptop**

```  
curl 172.17.4.99:38080/changeagent/changes
# [{"_id":2,"_ts":1467251967352398293,"data":{"Hello":"world"}}]
```
