import * as AuthSession from 'expo-auth-session';
import * as SecureStore from 'expo-secure-store';
import * as WebBrowser from 'expo-web-browser';
import { useEffect, useState } from 'react';
import { Button, SafeAreaView, Text } from 'react-native';
import { createApiClient } from '@__APP_SLUG__/api-client';
WebBrowser.maybeCompleteAuthSession();
const issuer=process.env.EXPO_PUBLIC_OIDC_ISSUER ?? 'http://localhost:5556/dex';
export default function Home(){const [token,setToken]=useState<string|null>(null);const discovery=AuthSession.useAutoDiscovery(issuer);const redirectUri=AuthSession.makeRedirectUri({scheme:'__APP_SLUG__',path:'callback'});const [request,response,prompt]=AuthSession.useAuthRequest({clientId:process.env.EXPO_PUBLIC_OIDC_CLIENT_ID ?? '__APP_SLUG__-mobile',redirectUri,scopes:['openid','profile','email'],usePKCE:true},discovery);useEffect(()=>{SecureStore.getItemAsync('access_token').then(setToken)},[]);useEffect(()=>{if(response?.type==='success'){const code=response.params.code;AuthSession.exchangeCodeAsync({clientId:process.env.EXPO_PUBLIC_OIDC_CLIENT_ID ?? '__APP_SLUG__-mobile',code,redirectUri,extraParams:{code_verifier:request?.codeVerifier ?? ''}},discovery!).then(r=>{SecureStore.setItemAsync('access_token',r.accessToken);setToken(r.accessToken)})}},[response]);return <SafeAreaView style={{padding:32,gap:16}}><Text style={{fontSize:32}}>__APP_NAME__</Text><Text>{token?'Authenticated':'Ready to sign in'}</Text><Button title={token?'Sign out':'Sign in'} onPress={()=>token?(SecureStore.deleteItemAsync('access_token'),setToken(null)):prompt()}/></SafeAreaView>}

