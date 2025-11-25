goose -dir ./migrations postgres "postgres://postgres:password@localhost:5432/PRmanager?sslmode=disable" status

goose -dir ./migrations postgres "postgres://postgres:password@localhost:5432/PRmanager?sslmode=disable" up 

goose -dir ./migrations postgres "postgres://postgres:password@localhost:5432/PRmanager?sslmode=disable" down

goose -dir ./migrations postgres "postgres://postgres:password@localhost:5432/PRmanager?sslmode=disable" up-by-one