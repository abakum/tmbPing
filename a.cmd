cd /d %~dp0
set NGROK_API_KEY=2O2fOVEM3spDIIkjbhawsdjVZBDF_5sbukrilkn7uH6EzxFake
set NGROK_AUTHTOKEN=2O2fC0Bbf9DawgvnlfUFCfkyvp_4S9MfmraRvNumMwFake
set web_addr=localhost:4040
(
echo version: "2"
echo web_addr: %web_addr%
echo authtoken: %NGROK_AUTHTOKEN%
echo api_key: %NGROK_API_KEY%
)>ngrok.yml 
start ngrok.exe http 8888 --config ngrok.yml 
TIMEOUT /T 2 /NOBREAK
del ngrok.yml
start http://%web_addr%

set bot=PingBot
set TOKEN=6103948400:AAGzdagrhtshstrqaMtoFake
set goBin=r:\PortableApps\tmbPing
go install
start "%bot%" %goBin%\tmbPing.exe 12391808474684Fake -1001787948229970Fake
