#./build/bin/geth --datadir ./data --mine
./rmdb.sh && ./generate-block-0.sh &&  ./start-test.sh

#rm -rf data/geth/* && ./build.sh && ./build/bin/geth --datadir ./data --mine --consensus dpos console  --unlock "0x45a6a925c9203e1d19e49aad489a5347dc655d11" --password "data.all/passwd.txt"

#rm -rf data/geth/* && ./build.sh && ./build/bin/geth --datadir ./data --mine --consensus dpos console  --unlock "0x45a6a925c9203e1d19e49aad489a5347dc655d11" --password "data.all/passwd.txt" >run1.log 2>&1
