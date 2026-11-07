
settings.log.file.set(true)
settings.log.file.path.set("/app/test_bestof.log")
settings.log.level.set(3)
settings.decoder.priorities.ffmpeg.set(1)
settings.decoder.priorities.mad.set(50)
settings.decoder.file_extensions.ffmpeg.set(list.filter(fun (m) -> m != "mp3" and m != "mp2" and m != "mpa", settings.decoder.file_extensions.ffmpeg()))
settings.decoder.mime_types.ffmpeg.set(list.filter(fun (m) -> m != "audio/mpeg" and m != "audio/MPA" and m != "application/octet-stream", settings.decoder.mime_types.ffmpeg()))
settings.decoder.decoders.set(["mad", "taglib", "flac", "ogg", "wav", "aac", "ffmpeg"])

s1 = input.http(id="stream_bestof", "https://streaming.hotmixradio.com/hotmixpro-best-of-pre")
output.dummy(s1)
