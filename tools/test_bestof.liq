
settings.log.file.set(false)
settings.log.stdout.set(true)
settings.log.level.set(3)
s1 = input.http(id='stream_bestof', 'https://streaming.hotmixradio.com/hotmixpro-best-of-pre')
output.dummy(s1)
