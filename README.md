```
make local-image

docker tag localhost:5000/scheduler-plugins/controller:latest zbsss/scheduler-plugins-controller:latest
docker tag localhost:5000/scheduler-plugins/kube-scheduler:latest zbsss/scheduler-plugins-kube-scheduler:latest

docker push zbsss/scheduler-plugins-controller:latest
docker push zbsss/scheduler-plugins-kube-scheduler:latest
```

```
cd scheduler-plugins/manifests/install/charts
helm install scheduler-plugins as-a-second-scheduler/ --create-namespace --namespace scheduler-plugins
```

```
kubectl get deploy -n scheduler-plugins

k apply -f manifests/sharedev/test.yaml

kind export logs
```
