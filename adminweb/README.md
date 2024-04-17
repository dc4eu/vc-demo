# Web demo
Browse: http://172.16.50.12:8080/

Username: `admin`

Password: `secret123`

Session inactivity (between request): 1 hour

### Upload
Type `authentic source person id` and select `document type`, then press `Upload` 
Uses a mocked authentic source payload

### Fetch
Type `authentic source person id` and select `document type`, then press `Fetch`
Uses /portal endpoint in apigw-service (datastore)

## Demo environment
### Run
`make start`

### Stop
`make stop`

### Restart
`make restart`





