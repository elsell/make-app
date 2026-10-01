import { useLayoutEffect, useMemo, useRef } from 'react';
import { createActionRegistry, type CommittedAction } from '@__APP_SLUG__/client-core';

type Presentation=Readonly<Record<string,string|number|boolean|null>>;
type HeaderAction=CommittedAction&{presentation:Presentation};
/** Pass a stable native option builder; only committed callbacks can execute. */
export function useNativeHeaderActionOptions<T>(actions:readonly HeaderAction[],build:(actions:readonly HeaderAction[])=>T):T{
 const registry=useRef<ReturnType<typeof createActionRegistry>|null>(null);
 if(!registry.current)registry.current=createActionRegistry();
 const current=registry.current;
 useLayoutEffect(()=>{current.commit(actions);return ()=>current.commit([]);},[actions,current]);
 const presentation=actions.map(({id,disabled,presentation})=>({id,disabled,presentation}));
 const signature=JSON.stringify(presentation);
 return useMemo(()=>build(presentation.map(action=>({...action,onPress:current.handler(action.id)}))),[signature,build,current]);
}
