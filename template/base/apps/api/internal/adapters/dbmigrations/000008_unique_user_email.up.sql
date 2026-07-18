CREATE UNIQUE INDEX user_models_normalized_email_unique_idx
ON user_models (lower(email))
WHERE email <> '';
