# accumulation-zone

Accumulation Zone is where glacier forms.

AWS Glacier is reliable, cost-effective and high-performant backup.

This application operates directly with AWS Glacier and sends incremental updates to cold storage. It uses local index file to track file changes and sends only updates.

## Backup and restore

Choose your option

### Backup and restore with docker

```shell
### Backup first pass
docker run -it -v /path/to/your/data:/data mrdunski/accumulation-zone changes upload \
  --account-id=... \
  --region-id=... \
  --vault-name=... \
  --key-id=... \
  --key-secret=...

### Backup incremental changes
docker run -it -v /path/to/your/data:/data mrdunski/accumulation-zone changes upload \
  --account-id=... \
  --region-id=... \
  --vault-name=... \
  --key-id=... \
  --key-secret=...

### ???

### 💣

### Recover from backup
docker run -it -v /path/to/your/data:/data mrdunski/accumulation-zone recover all \
  --account-id=... \
  --region-id=... \
  --vault-name=... \
  --key-id=... \
  --key-secret=... \
  --tier=Expedited
```

for more info use:

```shell
docker run -it mrdunski/accumulation-zone --help
```

### Backup and restore with command line

```shell
go build -o accumulation-zone

export VAULT_NAME=...
export ACCOUNT_ID=...
export REGION_ID=...
export KEY_ID=...
export KEY_SECRET=...
export PATH_TO_BACKUP=/path/to/your/data

### Backup first pass
./accumulation-zone changes upload

### Backup second pass
./accumulation-zone changes upload

### ???

### 💣

### Recover from backup
./accumulation-zone recover all --tier=Expedited
```

for more info use:

```shell
go build -o accumulation-zone && ./accumulation-zone --help
```

## Run unit tests

```shell
go test ./...
```

or use ginkgo

```shell
go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo@latest
ginkgo -r
```