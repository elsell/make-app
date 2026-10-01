#!/usr/bin/env python3
"""Validate production inputs without printing credentials."""
import os
import re
import sys
from urllib.parse import urlsplit,parse_qs

def validate(env):
 errors=[];prefix='__ENV_PREFIX___'
 def value(name):return env.get(prefix+name,'').strip()
 for name in ['API_IMAGE','WEB_IMAGE']:
  if not re.fullmatch(r'[a-zA-Z0-9][a-zA-Z0-9._:/-]*@sha256:[0-9a-f]{64}',value(name)):errors.append(name+' must be digest-pinned')
 for name in ['API_URL','WEB_URL','OIDC_ISSUER']:
  try:
   url=urlsplit(value(name))
   if url.scheme!='https' or not url.hostname or url.username or url.query or url.fragment:raise ValueError()
  except ValueError:errors.append(name+' must be an HTTPS URL')
 users=[]
 for name in ['MIGRATION_DSN','DATABASE_DSN']:
  try:
   url=urlsplit(value(name));ssl=parse_qs(url.query).get('sslmode',[''])[0]
   if url.scheme not in ('postgres','postgresql') or not url.hostname or not url.username or not url.password or ssl != 'verify-full':raise ValueError()
   users.append(url.username)
  except ValueError:errors.append(name+' requires a TLS PostgreSQL URL with credentials')
 if len(users)==2 and users[0]==users[1]:errors.append('migration and runtime database users must differ')
 for name in ['SPICEDB_SCHEMA_TOKEN','SPICEDB_TOKEN','CURSOR_SIGNING_KEY','METRICS_BEARER_TOKEN']:
  if len(value(name))<32 or 'change-me' in value(name) or 'development' in value(name):errors.append(name+' requires an independent production secret')
 if value('SPICEDB_SCHEMA_TOKEN')==value('SPICEDB_TOKEN'):errors.append('schema and runtime SpiceDB credentials must differ')
 for name in ['SPICEDB_UPSTREAM_ENDPOINT','WEB_OIDC_CLIENT_ID','OIDC_AUDIENCES']:
  if not value(name):errors.append(name+' is required')
 return errors
if __name__=='__main__':
 errors=validate(os.environ)
 for error in errors:print(error,file=sys.stderr)
 sys.exit(bool(errors))
