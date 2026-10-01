export function createPullLifecycle(present: (active: boolean) => void) {
 let focused=false;let sequence=0;let active:number|undefined;
 return {
  focus(){focused=true;},
  blur(){focused=false;sequence++;if(active!==undefined){active=undefined;present(false);}},
  async run(refresh:()=>Promise<void>){
   if(!focused||active!==undefined)return;
   const request=++sequence;active=request;present(true);
   try{await refresh();}finally{if(active===request){active=undefined;present(false);}}
  },
 };
}
export type CommittedAction={id:string;disabled?:boolean;onPress:()=>void};
export function createActionRegistry(){
 let current=new Map<string,CommittedAction>();
 return {
  commit(actions:readonly CommittedAction[]){
   const next=new Map(actions.map(action=>[action.id,action]));
   if(next.size!==actions.length)throw new Error('Native action identities must be unique');
   current=next;
  },
  handler(id:string){return ()=>{const action=current.get(id);if(action&&!action.disabled)action.onPress();};},
 };
}
