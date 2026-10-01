#!/usr/bin/env python3
import datetime
import os
import plistlib
import sys

def validate(profile,team,bundle,now):
 entitlements=profile.get('Entitlements',{})
 if profile.get('TeamIdentifier')!=[team] or entitlements.get('application-identifier')!=team+'.'+bundle:
  raise ValueError('profile identity mismatch')
 if entitlements.get('get-task-allow') is not False or profile.get('ProvisionedDevices') or profile.get('ProvisionsAllDevices'):
  raise ValueError('App Store distribution profile required')
 expiration=profile.get('ExpirationDate')
 if not isinstance(expiration,datetime.datetime) or expiration<=now:raise ValueError('expired profile')
 if not profile.get('DeveloperCertificates'):raise ValueError('distribution certificate missing')
if __name__=='__main__':
 try:validate(plistlib.load(sys.stdin.buffer),os.environ['APPLE_TEAM_ID'],os.environ['APPLE_BUNDLE_ID'],datetime.datetime.now(datetime.timezone.utc).replace(tzinfo=None))
 except Exception:raise SystemExit('Provisioning profile does not match the release') from None
