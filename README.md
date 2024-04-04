### Installation
```bash
go install github.com/nerrorsec/internetDB@latest
```

### Usage
```bash
# view all open ports
internetDB -r "8.8.8.8/24"
# check for specific ports
internetDB -r "8.8.8.8/24" -p 80,8080
# increased threads, rate limit applies
internetDB -r "8.8.8.8/24" -t 20
```
