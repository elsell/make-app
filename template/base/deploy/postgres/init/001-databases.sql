CREATE ROLE app LOGIN PASSWORD 'app';
CREATE ROLE app_audit_retention LOGIN PASSWORD 'app_audit_retention';
CREATE ROLE spicedb LOGIN PASSWORD 'spicedb';
ALTER SCHEMA public OWNER TO app_migrator;
GRANT ALL ON SCHEMA public TO app_migrator;
GRANT CONNECT ON DATABASE app TO app;
GRANT CONNECT ON DATABASE app TO app_audit_retention;
CREATE DATABASE spicedb OWNER spicedb;
