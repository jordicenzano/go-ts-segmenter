# go-ts-segmenter

#Tests snippets
Create 100m live stream with 3 advanced segments to disc (LHLS)
```
ffmpeg -f lavfi -re -i smptebars=duration=6000:size=320x200:rate=30 -f lavfi -i sine=frequency=1000:duration=6000:sample_rate=48000 -pix_fmt yuv420p -c:v libx264 -b:v 180k -g 60 -keyint_min 60 -profile:v baseline -preset veryfast -c:a libfdk_aac -b:a 96k -vf "drawtext=fontfile=/Library/Fonts/Arial.ttf: text=\'Local time %{localtime\: %Y\/%m\/%d %H.%M.%S} (%{n})\': x=10: y=10: fontsize=16: fontcolor=white: box=1: boxcolor=0x00000099" -f mpegts - | bin/manifest-generator -p ./results/ffmpeg -l 3 -p ./results/myfolder
```

Create 100m live stream with to disc with plain HLS
```
ffmpeg -f lavfi -re -i smptebars=duration=6000:size=320x200:rate=30 -f lavfi -i sine=frequency=1000:duration=6000:sample_rate=48000 -pix_fmt yuv420p -c:v libx264 -b:v 180k -g 60 -keyint_min 60 -profile:v baseline -preset veryfast -c:a libfdk_aac -b:a 96k -vf "drawtext=fontfile=/Library/Fonts/Arial.ttf: text=\'Local time %{localtime\: %Y\/%m\/%d %H.%M.%S} (%{n})\': x=10: y=10: fontsize=16: fontcolor=white: box=1: boxcolor=0x00000099" -f mpegts - | bin/manifest-generator -p ./results/ffmpeg -p ./results/myfolder
```