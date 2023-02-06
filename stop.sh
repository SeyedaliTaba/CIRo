./scion.sh stop
docker stop bazel-remote-cache
bazel shutdown
rm -rf gen/
rm -rf logs/
rm -rf gen-cache/
rm -rf traces/
