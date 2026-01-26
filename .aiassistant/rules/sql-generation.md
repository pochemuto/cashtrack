---
apply: always
---

# SQL Modification Guideline

Store all sql queries in queries file db/query.sql. The file is in the sqlc format (https://sqlc.dev/)
The schema is stored in db/schema.sql. Don't edit this file directly. Add a migration to new db/migrations/ file and 
run `make generate`. It will generate schema.sql and related go methods to query database. 