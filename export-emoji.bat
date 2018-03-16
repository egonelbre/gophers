@set ASESPRITE=f:\Games\Steam\steamapps\common\Aseprite\Aseprite.exe

%ASESPRITE% -b icon/emoji.ase -scale 3 --format json-array --sheet .thumb/icon/emoji-3x.png --data ~rendered\emoji-3x.json --sheet-type rows -sheet-width 672
go run twitterify.go .thumb/icon/emoji-3x.png ~rendered\emoji-3x-twitter.png

%ASESPRITE% -b icon/emoji.ase -scale 3 --save-as ~rendered/emoji-3x/gopher-{tag}-{frame}.png

%ASESPRITE% -b icon/emoji.ase --format json-array --sheet icon/emoji.png --data icon/emoji.json --sheet-type rows -sheet-width 224

%ASESPRITE% -b icon/emoji.ase --save-as icon/emoji/gopher-{tag}.png{frame}

pushd icon\emoji
del *.png
ren *.png0 *.png
popd

go run normalize-alpha.go icon/emoji/*.png