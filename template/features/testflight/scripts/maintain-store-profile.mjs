import { createPrivateKey } from 'node:crypto';
import { readFile, writeFile } from 'node:fs/promises';
import { appleMaintenanceClient } from './apple-maintenance-client.mjs';
import { inspectProfileRepair, repairPushProfile } from './ios-profile-maintenance.mjs';

try {
 const [mode, metadataPath, outputPath] = process.argv.slice(2);
 if(!['inspect','enable-push'].includes(mode)||!metadataPath)throw new Error('Invalid arguments');
 const profile=JSON.parse(await readFile(metadataPath,'utf8'));
 const expected={teamId:process.env.APPLE_TEAM_ID,bundleId:process.env.APPLE_BUNDLE_ID};
 const privateKey=createPrivateKey(Buffer.from(process.env.APP_STORE_CONNECT_API_KEY_BASE64??'','base64'));
 const api=appleMaintenanceClient({privateKey,keyId:process.env.APP_STORE_CONNECT_KEY_ID,issuerId:process.env.APP_STORE_CONNECT_ISSUER_ID});
 const plan=await inspectProfileRepair(api,profile,expected,new Date());
 if(mode==='enable-push') {
  if(!outputPath)throw new Error('Output path required');
  const response=await repairPushProfile(api,plan,'__APP_NAME__ Push');
  const content=response?.data?.attributes?.profileContent;
  if(typeof content!=='string'||!content)throw new Error('Missing replacement profile');
  await writeFile(outputPath,Buffer.from(content,'base64'),{mode:0o600,flag:'wx'});
 }
 process.stdout.write(JSON.stringify({pushEnabled:plan.pushEnabled,existingCertificateMatched:true})+'\n');
} catch {process.stderr.write('Profile maintenance failed; existing credentials were not rotated.\n');process.exitCode=1;}
