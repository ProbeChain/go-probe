./build.sh && nohup ./build/bin/geth --log.debug  --verbosity 5  --datadir ./data --networkid 2016  --cache 512 --http --http.corsdomain '*'  --allow-insecure-unlock  --nodiscover  --mine --miner.threads 1  --unlock "0x45A6A925C9203e1D19e49AaD489a5347dc655D11" --password "data.all/passwd.txt" >run1.log 2>&1 &