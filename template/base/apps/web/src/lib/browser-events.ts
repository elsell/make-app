/** Browser capabilities are isolated here; messages are untrusted refetch hints. */
export function browserEvents(channelName: string) {
 return {
  addFocus(listener:()=>void){window.addEventListener('focus',listener);},
  removeFocus(listener:()=>void){window.removeEventListener('focus',listener);},
  addVisibility(listener:()=>void){document.addEventListener('visibilitychange',listener);},
  removeVisibility(listener:()=>void){document.removeEventListener('visibilitychange',listener);},
  visible:()=>document.visibilityState==='visible',
  openChannel(receive:(value:unknown)=>void){
   const channel=new BroadcastChannel(channelName);
   const listener=(event:MessageEvent<unknown>)=>receive(event.data);
   try{channel.addEventListener('message',listener);}catch(error){channel.close();throw error;}
   return {publish(value:unknown){channel.postMessage(value);},close(){channel.removeEventListener('message',listener);channel.close();}};
  },
 };
}
