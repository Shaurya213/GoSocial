# GoSocial

## Setup
Setup Docker using the following command

```docker-compose -f docker-compose.yml up -d````

Run the http media server

```go run ./cmd/media-server/```

Pick and choose the service you want

Chats
```go run ./cmd/chat-svc/```

User
```go run ./cmd/user-svc/```

Notifications
```go run ./cmd/notifs-svc/```

Notifications
```go run ./cmd/feed-svc/```

## How to connect setup for cloud server setup

In LoadConfig() in config.go, you can uncomment these lines and have the config get the current ip of your VM and add these in your config so that connecting through to an external mongo or sql or media-server setup dosen't require changing the .env everytime.

```func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil{
		log.Fatalf(".env is not laoding: %v", err)
	}

	cmd := exec.Command("bash", "-c", "curl ifconfig.me")
	out, _ := cmd.Output()
	ip:= string(out)
	//ip = "localhost"

	//Setting new envs
	//os.Setenv("MONGO_HOST", ip)
	//os.Setenv("MYSQL_HOST", ip)
```

# Chats
## Working chat streaming
<img width="3072" height="1728" alt="image" src="https://github.com/user-attachments/assets/7cfa18cb-6204-4056-8214-0c4c306d4793" />

