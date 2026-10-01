import { useCallback, useRef, useState } from 'react';
import { useFocusEffect } from 'expo-router';
import { createPullLifecycle } from '@__APP_SLUG__/client-core';

/** Connect refreshing/onRefresh to the native refresh control; no background query flags. */
export function usePullRefresh(refreshData:()=>Promise<void>){
 const [refreshing,setRefreshing]=useState(false);
 const lifecycle=useRef<ReturnType<typeof createPullLifecycle>|null>(null);
 if(!lifecycle.current)lifecycle.current=createPullLifecycle(setRefreshing);
 const pull=lifecycle.current;
 useFocusEffect(useCallback(()=>{pull.focus();return ()=>pull.blur();},[pull]));
 return {refreshing,refresh:()=>pull.run(refreshData)};
}
