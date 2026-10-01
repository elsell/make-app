#!/usr/bin/env python3
"""Package only reviewed, non-secret deployment inputs."""
import argparse
import hashlib
from pathlib import Path
import re
import tarfile
import tempfile

INPUTS=('deploy/selfhost/compose.yaml','deploy/selfhost/environment.example','scripts/selfhost-preflight.py','docs/self-hosting.md')
def build(root,out,version,api,web):
 if not re.fullmatch(r'v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)',version):raise ValueError('stable SemVer required')
 for value in (api,web):
  if not re.fullmatch(r'[a-zA-Z0-9][a-zA-Z0-9._:/-]*@sha256:[0-9a-f]{64}',value):raise ValueError('digest-pinned image required')
 out.mkdir(parents=True,exist_ok=True);archive=out/'__APP_SLUG__-selfhost.tar.gz'
 with tempfile.TemporaryDirectory() as d:
  stage=Path(d)
  for name in INPUTS:
   source=root/name
   if source.is_symlink():raise ValueError('symlinked bundle input refused')
   target=stage/name;target.parent.mkdir(parents=True,exist_ok=True);target.write_bytes(source.read_bytes())
  p=stage/'deploy/selfhost/environment.example'
  p.write_text(p.read_text().replace('__ENV_PREFIX___API_IMAGE=','__ENV_PREFIX___API_IMAGE='+api).replace('__ENV_PREFIX___WEB_IMAGE=','__ENV_PREFIX___WEB_IMAGE='+web))
  (stage/'VERSION').write_text(version+'\n')
  with tarfile.open(archive,'w:gz') as tar:
   for p in sorted(stage.rglob('*')):
    if p.is_file():tar.add(p,arcname=str(p.relative_to(stage)),recursive=False)
 digest=hashlib.sha256(archive.read_bytes()).hexdigest();archive.with_suffix(archive.suffix+'.sha256').write_text(digest+'  '+archive.name+'\n')
 return archive
if __name__=='__main__':
 parser=argparse.ArgumentParser();parser.add_argument('--version',required=True);parser.add_argument('--api-image',required=True);parser.add_argument('--web-image',required=True);parser.add_argument('--output',type=Path,default=Path('release-dist'));args=parser.parse_args()
 build(Path(__file__).resolve().parents[1],args.output,args.version,args.api_image,args.web_image)
