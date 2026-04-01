ALTER DATABASE "{{.DatabaseName}}" SET "app.settings.jwt_secret" TO '{{.JWTSecret}}';
ALTER DATABASE "{{.DatabaseName}}" SET "app.settings.jwt_exp" TO '3600';
