#./build/bin/gprobe --datadir ./data --mine
./rmdb.sh && ./generate-block-0.sh &&  ./start-test.sh

#rm -rf data/gprobe/* && ./build.sh && ./build/bin/gprobe --datadir ./data --mine --consensus dpos console  --unlock "0x45a6a925c9203e1d19e49aad489a5347dc655d11" --password "data.all/passwd.txt"

#rm -rf data/gprobe/* && ./build.sh && ./build/bin/gprobe --datadir ./data --mine --consensus dpos console  --unlock "0x45a6a925c9203e1d19e49aad489a5347dc655d11" --password "data.all/passwd.txt" >run1.log 2>&1
