CREATE TABLE rate_limit_shard_models (
    scope text NOT NULL,
    shard integer NOT NULL CHECK (shard >= 0 AND shard < 64),
    PRIMARY KEY (scope, shard)
);
INSERT INTO rate_limit_shard_models(scope, shard)
SELECT scope, shard FROM (VALUES ('audit'), ('request')) AS scopes(scope)
CROSS JOIN generate_series(0, 63) AS shard;

CREATE TABLE rate_limit_window_models (
    scope text NOT NULL,
    principal_hash text NOT NULL,
    shard integer NOT NULL CHECK (shard >= 0 AND shard < 64),
    window_start timestamptz NOT NULL,
    request_count integer NOT NULL CHECK (request_count > 0),
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (scope, principal_hash),
    FOREIGN KEY (scope, shard) REFERENCES rate_limit_shard_models(scope, shard) ON DELETE CASCADE
);
CREATE INDEX rate_limit_window_eviction_idx ON rate_limit_window_models(scope, shard, updated_at);
GRANT SELECT, INSERT, UPDATE, DELETE ON rate_limit_window_models TO app;
GRANT SELECT, UPDATE ON rate_limit_shard_models TO app;
