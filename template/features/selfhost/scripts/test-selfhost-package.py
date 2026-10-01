import importlib.util
from pathlib import Path
import tempfile
import tarfile
import unittest
spec=importlib.util.spec_from_file_location('package',Path(__file__).with_name('build-selfhost-release.py'));module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
class Packaging(unittest.TestCase):
 def test_only_reviewed_inputs_are_shipped(self):
  with tempfile.TemporaryDirectory() as d:
   root=Path(d);(root/'deploy/selfhost').mkdir(parents=True);(root/'scripts').mkdir()
   for name in module.INPUTS:
    p=root/name;p.parent.mkdir(parents=True,exist_ok=True);p.write_text('safe')
   (root/'.env').write_text('PRIVATE_SECRET=secret')
   out=root/'output';a='registry.example/app@sha256:'+'a'*64;b='registry.example/web@sha256:'+'b'*64
   archive=module.build(root,out,'v1.2.3',a,b)
   with tarfile.open(archive) as tar:
    self.assertNotIn('.env',tar.getnames());self.assertIn('VERSION',tar.getnames())
   with self.assertRaises(ValueError):module.build(root,out,'../escape',a,b)
   with self.assertRaises(ValueError):module.build(root,out,'v1.2.3','app:latest',b)
if __name__=='__main__':unittest.main()
