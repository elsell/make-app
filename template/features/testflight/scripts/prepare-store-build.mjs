import { readFile, writeFile } from 'node:fs/promises';
import { pathToFileURL } from 'node:url';

export function prepare(config, tag, build) {
 if(!/^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/.test(tag)||!/^[1-9]\d{0,3}(?:\.[1-9]\d?){0,2}$/.test(build))throw new Error('Stable release tag and positive build number required');
 return {...config,expo:{...config.expo,version:tag.slice(1),ios:{...config.expo.ios,buildNumber:build},plugins:config.expo.plugins.filter(plugin=>(Array.isArray(plugin)?plugin[0]:plugin)!=='expo-dev-client')}};
}
if(process.argv[1] && import.meta.url===pathToFileURL(process.argv[1]).href){
 const file='apps/mobile/app.json';const [tag,build]=process.argv.slice(2);
 await writeFile(file,JSON.stringify(prepare(JSON.parse(await readFile(file,'utf8')),tag,build),null,2)+'\n');
}
