import importlib.util
from pathlib import Path
import unittest
spec=importlib.util.spec_from_file_location('preflight',Path(__file__).with_name('selfhost-preflight.py'));module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
class Preflight(unittest.TestCase):
 def test_credentials_and_tls_match_runtime(self):
  values={'API_IMAGE':'example.com/api@sha256:'+'a'*64,'WEB_IMAGE':'example.com/web@sha256:'+'b'*64,'API_URL':'https://api.example.com','WEB_URL':'https://web.example.com','OIDC_ISSUER':'https://login.example.com','MIGRATION_DSN':'postgres://migration:secret@db.example/app?sslmode=verify-full','DATABASE_DSN':'postgres://runtime:secret@db.example/app?sslmode=verify-full','SPICEDB_UPSTREAM_ENDPOINT':'auth.example:443','SPICEDB_SCHEMA_TOKEN':'a'*32,'SPICEDB_TOKEN':'b'*32,'CURSOR_SIGNING_KEY':'c'*32,'METRICS_BEARER_TOKEN':'d'*32,'WEB_OIDC_CLIENT_ID':'web','OIDC_AUDIENCES':'web,mobile'}
  env={'__ENV_PREFIX___'+k:v for k,v in values.items()};self.assertEqual(module.validate(env),[])
  for key,value in [('DATABASE_DSN',values['MIGRATION_DSN']),('DATABASE_DSN',values['DATABASE_DSN'].replace('verify-full','verify-ca')),('SPICEDB_TOKEN','a'*32),('API_IMAGE','api:latest'),('OIDC_ISSUER','http://login.example.com')]:
   with self.subTest(key=key,value=value):self.assertTrue(module.validate({**env,'__ENV_PREFIX___'+key:value}))
if __name__=='__main__':unittest.main()
