import datetime
import importlib.util
from pathlib import Path
import unittest
spec=importlib.util.spec_from_file_location('profile',Path(__file__).with_name('verify-store-profile.py'));module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
class Profile(unittest.TestCase):
 def test_scope_expiry_and_distribution(self):
  now=datetime.datetime(2026,1,1);p={'TeamIdentifier':['TEAM'],'Entitlements':{'application-identifier':'TEAM.com.example.app','get-task-allow':False},'ExpirationDate':now+datetime.timedelta(days=10),'DeveloperCertificates':[b'certificate']}
  module.validate(p,'TEAM','com.example.app',now)
  for change in [{'TeamIdentifier':['OTHER']},{'ExpirationDate':now},{'ProvisionedDevices':['phone']},{'DeveloperCertificates':[]}]:
   with self.assertRaises(ValueError):module.validate({**p,**change},'TEAM','com.example.app',now)
if __name__=='__main__':unittest.main()
