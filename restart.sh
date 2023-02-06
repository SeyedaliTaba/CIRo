./scion.sh stop
docker stop bazel-remote-cache
bazel shutdown
#rm -rf gen/
rm -rf logs/*
rm -rf gen-cache/*
rm -rf traces/data/*
rm -rf traces/key/*
./scion.sh bazel_remote
#./scion.sh topology -c topology/scionlab.topo
#for i in $(find -name "cs*1.toml");
#do
#	printf "\n[beaconing]\n  origination_interval = \"10m\"\n  propagation_interval = \"10m\"\n  registration_interval = \"10m\"\n carbon_forecast_interval=\"60m\"" >> $i;
#done
#./specify_zones.sh
#./write_electricity_maps_api_key.sh
./scion.sh run

