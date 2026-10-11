import argparse, json, os, time, subprocess, signal, tempfile, threading, urllib.request, urllib.parse, http.cookiejar, resource, sys, socket
from http.server import ThreadingHTTPServer, BaseHTTPRequestHandler
from pathlib import Path
parser=argparse.ArgumentParser(description="Local synthetic ArrScope sync and HTTP load check; never uses real services.")
parser.add_argument("binary",type=Path,help="Path to a built ArrScope binary")
parser.add_argument("--files",type=int,default=50000)
parser.add_argument("--emby",action="store_true",help="Supplement per-track metadata through a synthetic Emby API")
args=parser.parse_args(); binary=args.binary.resolve(); N=args.files
assert N > 0
def free_port():
 with socket.socket() as sock:
  sock.bind(('127.0.0.1',0)); return sock.getsockname()[1]
app_port=free_port()
class Fixture(BaseHTTPRequestHandler):
 def log_message(self,*a): pass
 def do_GET(self):
  u=urllib.parse.urlparse(self.path); q=urllib.parse.parse_qs(u.query)
  assert self.headers.get('X-Emby-Token' if u.path.startswith('/emby/') else 'X-Api-Key')=='synthetic-test-key'
  assert 'api_key' not in q and 'apikey' not in q
  if u.path=='/api/v3/movie': data=[{'id':i,'title':f'Movie {i:05d}','year':2020} for i in range(1,N+1)]
  elif u.path=='/api/v3/moviefile':
   data=[{'id':int(i),'movieId':int(i),'relativePath':f'Movie.{int(i):05d}.2020.1080p.WEB-DL.x264.mkv','size':4000000000,'quality':{'quality':{'name':'WEBDL-1080p'}},'mediaInfo':{'videoCodec':'AVC'}} for i in q['movieId']]
  elif u.path=='/emby/Items':
   start=int(q['StartIndex'][0]);limit=int(q['Limit'][0])
   items=[{'Id':str(i),'Type':'Movie','MediaSources':[{'Path':f'/media/Movie.{i:05d}.2020.1080p.WEB-DL.x264.mkv','Size':4000000000,'MediaStreams':[{'Type':'Audio','Index':1,'Codec':'dts','Language':'ger','Channels':6,'ChannelLayout':'5.1','IsDefault':True},{'Type':'Audio','Index':2,'Codec':'ac3','Language':'eng','Channels':6,'ChannelLayout':'5.1'},{'Type':'Video','Index':0,'ExtendedVideoType':'DolbyVision','ExtendedVideoSubType':'Hdr10'}]}]} for i in range(start+1,min(start+limit,N)+1)]
   data={'Items':items,'TotalRecordCount':N}
  elif u.path=='/emby/System/Info': data={'Version':'fixture'}
  else:
   assert u.path=='/api/v3/system/status',u.path
   data={'version':'fixture'}
  raw=json.dumps(data).encode();self.send_response(200);self.send_header('Content-Type','application/json');self.send_header('Content-Length',len(raw));self.end_headers();self.wfile.write(raw)
fixture=ThreadingHTTPServer(('127.0.0.1',0),Fixture);threading.Thread(target=fixture.serve_forever,daemon=True).start()
jar=http.cookiejar.CookieJar();opener=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar));base=f'http://127.0.0.1:{app_port}';csrf=''
def req(path,data=None,method=None):
 headers={'Content-Type':'application/json','X-CSRF-Token':csrf}
 r=urllib.request.Request(base+path,data=None if data is None else json.dumps(data).encode(),headers=headers,method=method)
 with opener.open(r,timeout=30) as response:return json.load(response)
with tempfile.TemporaryDirectory(prefix='arrscope-load-') as appdata:
 env={**os.environ,'ARRSCOPE_DATA_DIR':appdata,'ARRSCOPE_LISTEN':f'127.0.0.1:{app_port}','GOMAXPROCS':'2','GOMEMLIMIT':'96MiB'}
 env.pop('ARRSCOPE_DEMO',None)
 proc=subprocess.Popen([str(binary)],env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)
 try:
  for _ in range(100):
   try:req('/healthz');break
   except Exception:time.sleep(.05)
  req('/api/setup',{'username':'load-test','password':'Synthetic-load-test-2026','token':Path(appdata,'setup-token').read_text().strip()})
  csrf=req('/api/status')['csrf']
  cfg=req('/api/settings'); cfg['radarr'].update(enabled=True,url=f'http://127.0.0.1:{fixture.server_port}',apiKey='synthetic-test-key',interval=0)
  if args.emby: cfg['emby'].update(enabled=True,url=f'http://127.0.0.1:{fixture.server_port}',apiKey='synthetic-test-key',interval=0)
  start=time.perf_counter();req('/api/settings',cfg,'PUT')
  for _ in range(600):
   status=req('/api/status')['sync']['radarr']
   if not status['running'] and status['count']==N:break
   if not status['running'] and status['error']:raise RuntimeError(status['error'])
   time.sleep(.05)
  assert status['count']==N,status
  sync_seconds=time.perf_counter()-start
  timings=[]
  for _ in range(60):
   t=time.perf_counter();result=req('/api/files?service=radarr&resolution=1080p&codec=x264');timings.append((time.perf_counter()-t)*1000)
   assert result['total']==N and len(result['files'])==100
   if args.emby: assert all(f['embyMatch']=='matched' and len(f['audioTracks'])==2 for f in result['files'])
  if args.emby:
   result=req('/api/files?service=radarr&audio=DTS&audio=AC-3&audioOp=all&language=Deutsch&language=Englisch&languageOp=all')
   assert result['total']==N
  time.sleep(1)
 finally:
  proc.send_signal(signal.SIGTERM)
  _, exit_status, usage=os.wait4(proc.pid,0)
  proc.returncode=os.waitstatus_to_exitcode(exit_status)
  out,err=proc.communicate()
  peak=usage.ru_maxrss / ((1024*1024) if sys.platform=='darwin' else 1024)
 # Restart the same 50k cache and exercise a page read; collect its own peak RSS.
 proc=subprocess.Popen([str(binary)],env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)
 try:
  for _ in range(100):
   try:req('/healthz');break
   except Exception:time.sleep(.05)
  assert req('/api/settings')['radarr']['hasKey']
  assert req('/api/files?service=radarr&resolution=1080p&codec=x264')['total']==N
  time.sleep(1)
 finally:
  proc.send_signal(signal.SIGTERM)
  _, exit_status, idle_usage=os.wait4(proc.pid,0)
  proc.returncode=os.waitstatus_to_exitcode(exit_status)
  proc.communicate()
  idle_peak=idle_usage.ru_maxrss / ((1024*1024) if sys.platform=='darwin' else 1024)
 fixture.shutdown()
 print(json.dumps({'files':N,'emby':args.emby,'sync_seconds':round(sync_seconds,3),'http_filter_p95_ms':round(sorted(timings)[57],3),'platform':f'{sys.platform}, GOMAXPROCS=2, native build','peak_rss_mib':round(peak,2),'idle_restart_peak_rss_mib':round(idle_peak,2),'exit_code':proc.returncode},indent=2))
