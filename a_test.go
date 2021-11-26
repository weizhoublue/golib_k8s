package golib_k8s_test

import (
	"fmt"
	k8s "golib_k8s"
	"testing"
	"time"

	// for creating deployment
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	cache "k8s.io/client-go/tools/cache"
)







func Test_1(t *testing.T){

	k8s.EnableLog=true
	k:=k8s.K8sClient{}


	namespace:="default"
	if poddata, err:=k.ListPods(namespace) ; err!=nil {
		fmt.Println(  err )
		t.FailNow()
	}else{
		fmt.Printf("podInfo=%v \n" , poddata)
	}

	podName:="test3-nginx-585dbc7464-txjl6"
	if b , err :=k.CheckPodHealthy( namespace  ,  podName ) ; err !=nil {
		fmt.Println(  err )
		t.FailNow()
	}else{
		fmt.Printf("healthy=%v \n" , b )
	}

}


func Test_hostPod(t *testing.T){

	k8s.EnableLog=false
	k:=k8s.K8sClient{}


	namespace:=""
	if podlist , err:=k.ListPods(namespace) ; err!=nil {
		fmt.Println(  err )
		t.FailNow()
	}else{
		//fmt.Printf("podInfo=%v \n" , podInfo)
		for k , v :=range podlist {
			fmt.Printf("pod %v : HostNetwork=%v NodeName=%v PodIP=%v Phase=%v podName=%v \n" , 
				  k,   v.Spec.HostNetwork   ,  v.Spec.NodeName , v.Status.PodIP , v.Status.Phase , v.ObjectMeta.Name )
		}
	}


}


func Test_node(t *testing.T){

	k8s.EnableLog=false
	k:=k8s.K8sClient{}

	if info , _ , err:=k.GetNodes( ) ; err!=nil {
		fmt.Println(  err )
		t.FailNow()
	}else{
		//fmt.Printf("podInfo=%v \n" , podInfo)
		for n , v :=range info {
			fmt.Printf("node %v :  %v \n" ,  n, v )
		}
	}


}



// recommended method
func Test_2(t *testing.T){

	k8s.EnableLog=true
	k:=k8s.K8sClient{}



	var namespace string


	//===============  create ==============
	namespace="default"
	deploymentName:="demo-deployment"
	// https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured#Unstructured
	// wirte it same with yaml spec
	deploymentYaml:=&unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name": deploymentName ,
			},
			"spec": map[string]interface{}{
				"replicas": 2,
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": "demo",
					},
				},
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "demo",
						},
					},

					"spec": map[string]interface{}{
						"containers": []map[string]interface{}{
							{
								"name":  "web",
								"image": "nginx:1.12",
								"ports": []map[string]interface{}{
									{
										"name":          "http",
										"protocol":      "TCP",
										"containerPort": 80,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if err:=k.CreateDeployment( namespace,  deploymentYaml ) ; err!=nil {
		fmt.Println(  err )
		t.FailNow()
	}else{
		fmt.Printf("succeeded to create ")
	}
	time.Sleep(10*time.Second)




	//===============  update ==============
	//    You have two options to Update() this Deployment:
	//
	//    1. Modify the "deployment" variable and call: Update(deployment).
	//       This works like the "kubectl replace" command and it overwrites/loses changes
	//       made by other clients between you Create() and Update() the object.
	//    2. Modify the "result" returned by Get() and retry Update(result) until
	//       you no longer get a conflict error. This way, you can preserve changes made
	//       by other clients between Create() and Update(). This is implemented below
	//			 using the retry utility package included with client-go. (RECOMMENDED)
	//
	// More Info:
	// https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency
	namespace="default"
	deploymentName="demo-deployment"
	hander:=func( deploymentYaml *unstructured.Unstructured) error {
		// update replicas to 1
		// https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured
		if err := unstructured.SetNestedField(deploymentYaml.Object, int64(1), "spec", "replicas"); err != nil {
			return fmt.Errorf("failed to set replica value: %v", err)
		}

		//update containers
		// extract spec containers
		containers, found, err := unstructured.NestedSlice(deploymentYaml.Object, "spec", "template", "spec", "containers")
		if err != nil || !found || containers == nil {
			return fmt.Errorf("deployment containers not found or error in spec: %v", err)
		}

		// update container[0] image
		if err := unstructured.SetNestedField(containers[0].(map[string]interface{}), "nginx:1.13", "image"); err != nil {
			return  err
		}
		if err := unstructured.SetNestedField(deploymentYaml.Object, containers, "spec", "template", "spec", "containers"); err != nil {
			return err
		}
		return nil 
	}
	if err:=k.UpdateDeployment(namespace , deploymentName , hander ) ; err!=nil {
		fmt.Println(  err )
		t.FailNow()
	}
	fmt.Println("succeed to Updated deployment...")

	time.Sleep(10*time.Second)






	//===============  get ==============
	namespace="default"
	if deploymentDetailInfo , e:=k.ListDeployment( namespace  ) ; e!=nil {
		fmt.Println(  e )
		t.FailNow()
	}else{

		// show how to get information from unstructured.Unstructured (deploymentDetailInfo)
		for _ , v := range deploymentDetailInfo {
			// methods : https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured#Unstructured

			// call named method 
			name:=v.GetName() 
			fmt.Printf("deployment %s GetNamespace : %v \n" , name , v.GetNamespace() )
			fmt.Printf("deployment %s GetLabels : %v \n" , name , v.GetLabels() )

			//a general method to get whatever information you want from the deplyment yaml
			// unstructured:  https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured
			replicas, found, err := unstructured.NestedInt64( v.Object, "spec", "replicas")
			if err != nil || !found {
				fmt.Printf("Replicas not found for deployment %s: error=%s", v.GetName() , err)
				t.FailNow()
			}else{
				fmt.Printf("deployment %s replicas=%v  \n" , name , replicas )
			}

		}

	}
	time.Sleep(10*time.Second)




	//===============  delete ==============
	namespace="default"
	deploymentName="demo-deployment"
	if err:=k.DelDeployment( namespace  , deploymentName  ); err!=nil {
		fmt.Println(  err )
		t.FailNow()
	}
	fmt.Printf("succeeded to delete \n")


}





func Test_3(t *testing.T){


	k8s.EnableLog=true
	k:=k8s.K8sClient{}




	//=============== create deployment ============
	namespace:="default"
	deploymentName:="demo-deployment"

	replicasNum:=int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: deploymentName ,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicasNum ,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "web",
							Image: "nginx:1.12",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}	
	if err:=k.CreateDeploymentTyped( namespace , deployment ) ; err!=nil {
		fmt.Println(  err )
		t.FailNow()
	}

	fmt.Println("succeeded to create ")
	time.Sleep(10*time.Second)




	//===============  update ==============
	//    You have two options to Update() this Deployment:
	//
	//    1. Modify the "deployment" variable and call: Update(deployment).
	//       This works like the "kubectl replace" command and it overwrites/loses changes
	//       made by other clients between you Create() and Update() the object.
	//    2. Modify the "result" returned by Get() and retry Update(result) until
	//       you no longer get a conflict error. This way, you can preserve changes made
	//       by other clients between Create() and Update(). This is implemented below
	//			 using the retry utility package included with client-go. (RECOMMENDED)
	//
	// More Info:
	// https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency
	namespace="default"
	deploymentName="demo-deployment"
	hander:=func( deployment *appsv1.Deployment) error {

		a:=int32(1)
		deployment.Spec.Replicas = &a                           // reduce replica count
		deployment.Spec.Template.Spec.Containers[0].Image = "nginx:1.13" // change nginx version

		return nil 
	}
	if err:=k.UpdateDeploymentTyped(namespace , deploymentName , hander ) ; err!=nil {
		fmt.Println(  err )
		t.FailNow()
	}
	fmt.Println("succeed to Updated deployment...")

	time.Sleep(10*time.Second)





	// ================= get all deploy for all namespace ================
	namespace="default"
	if deploymentBasicInfo , err:=k.ListDeploymentTyped( namespace  ) ; err!=nil {
		fmt.Println(  err )
		t.FailNow()
	}else{
		fmt.Printf("deploymentBasicInfo=%v \n" , deploymentBasicInfo )
	}
	time.Sleep(10*time.Second)





	//=============== delete  =================
	fmt.Println("delete it  ")
	namespace="default"
	deploymentName="demo-deployment"
	if err:=k.DelDeploymentTyped( namespace , deploymentName  ) ; err!=nil {
		fmt.Println(  err )
		t.FailNow()
	}else{
		fmt.Printf("succeeded to delete  \n"  )

	}


}



func Test_4(t *testing.T){

/*  test

#资源类 和 子资源类 例子
kubectl create clusterrole test1-clusterrole -n default --verb=get,list --resource=deployments.apps,deployments.extensions --resource=pods/log --resource=pods 
kubectl create clusterrolebinding test11 -n default --clusterrole=test1-clusterrole  --user=jane
kubectl create clusterrolebinding test12 -n default --clusterrole=test1-clusterrole  --group=janeGroup
# kubectl delete clusterrole test1-clusterrole
# kubectl delete clusterrolebinding test11
# kubectl delete clusterrolebinding test12

#非资源类  例子
kubectl create clusterrole test2-clusterrole -n default --verb=get,post --non-resource-url=/health
kubectl create clusterrolebinding test21 -n default --clusterrole=test2-clusterrole --user=jane
kubectl create clusterrolebinding test22 -n default --clusterrole=test2-clusterrole --group=janeGroup
# kubectl delete clusterrole test2-clusterrole
# kubectl delete clusterrolebinding test21
# kubectl delete clusterrolebinding test22

#资源的实例 例子
kubectl create clusterrole test3-clusterrole -n default --verb=get --resource=configmaps --resource-name=tom-config  
kubectl create clusterrolebinding test31 -n default --clusterrole=test3-clusterrole  --user=jane
kubectl create clusterrolebinding test32 -n default --clusterrole=test3-clusterrole  --group=janeGroup
# kubectl delete clusterrole test3-clusterrole
# kubectl delete clusterrolebinding test31
# kubectl delete clusterrolebinding test32


*/

	k8s.EnableLog=true
	k:=k8s.K8sClient{}



	//========================== resource
	userName:="jane"
	userGroupName:=[]string{}

	checkVerb:=k8s.VerbGet
	checkResName:="deployments"
	checkSubResName:=""
	checkResInstanceName:=""
	checkResApiGroup:="apps"
	checkResNamespace:="default"

	if allowed , reason , err:= k.CheckUserRole(userName, userGroupName , 
		checkVerb, checkResName, checkSubResName , checkResInstanceName  , checkResApiGroup , checkResNamespace  ); err!=nil{
		fmt.Printf("error: %v \n" , err )
	}else {
		fmt.Printf("allowed?%v , reason: %s \n" ,allowed , reason )
	}


	//========================== resource subresource
	userName="jane"
	userGroupName=[]string{}

	checkVerb=k8s.VerbGet
	checkResName="pods"
	checkSubResName="log"
	checkResInstanceName=""
	checkResApiGroup=""
	checkResNamespace="default"

	if allowed , reason , err:= k.CheckUserRole(userName, userGroupName , 
		checkVerb, checkResName, checkSubResName , checkResInstanceName  , checkResApiGroup , checkResNamespace  ); err!=nil{
		fmt.Printf("error: %v \n" , err )
	}else {
		fmt.Printf("allowed?%v , reason: %s \n" ,allowed , reason )
	}


	//========================== resource instance
	userName="jane"
	userGroupName=[]string{}

	checkVerb=k8s.VerbGet
	checkResName="configmaps"
	checkSubResName=""
	checkResInstanceName="tom-config"
	checkResApiGroup=""
	checkResNamespace="default"

	if allowed , reason , err:= k.CheckUserRole(userName, userGroupName , 
		checkVerb, checkResName, checkSubResName , checkResInstanceName  , checkResApiGroup , checkResNamespace  ); err!=nil{
		fmt.Printf("error: %v \n" , err )
	}else {
		fmt.Printf("allowed?%v , reason: %s \n" ,allowed , reason )
	}



	//========================== noresource
	userName="jane"
	userGroupName=[]string{}
	checkVerb=k8s.VerbGet
	checkResName="/health"
	if allowed , reason , err:= k.CheckUserRole(userName, userGroupName , 
		checkVerb, checkResName, "" , ""  , "" , ""  ); err!=nil{
		fmt.Printf("error: %v \n" , err )
	}else {
		fmt.Printf("allowed?%v , reason: %s \n" ,allowed , reason )
	}


	userName=""
	userGroupName=[]string{"janeGroup"}
	checkVerb=k8s.VerbGet
	checkResName="/health"
	if allowed , reason , err:= k.CheckUserRole(userName, userGroupName , 
		checkVerb, checkResName, "" , ""  , "" , ""  ); err!=nil{
		fmt.Printf("error: %v \n" , err )
	}else {
		fmt.Printf("allowed?%v , reason: %s \n" ,allowed , reason )
	}


	userName="jane"
	userGroupName=[]string{"janeGroup"}
	checkVerb=k8s.VerbGet
	checkResName="/health"
	if allowed , reason , err:= k.CheckUserRole(userName, userGroupName , 
		checkVerb, checkResName, "" , ""  , "" , ""  ); err!=nil{
		fmt.Printf("error: %v \n" , err )
	}else {
		fmt.Printf("allowed?%v , reason: %s \n" ,allowed , reason )
	}



}






func Test_5(t *testing.T){


	k8s.EnableLog=true
	k:=k8s.K8sClient{}



	//========================== resource

	checkVerb:=k8s.VerbGet
	checkResName:="deployments"
	checkSubResName:=""
	checkResInstanceName:=""
	checkResApiGroup:="apps"
	checkResNamespace:="default"

	if allowed , reason , err:= k.CheckSelfRole( checkVerb, checkResName, checkSubResName , checkResInstanceName  , checkResApiGroup , checkResNamespace  ) ; err!=nil{
		fmt.Printf("error: %v \n" , err )
	}else {
		fmt.Printf("allowed?%v , reason: %s \n" ,allowed , reason )
	}


	//========================== resource subresource


	checkVerb=k8s.VerbGet
	checkResName="pods"
	checkSubResName="log"
	checkResInstanceName=""
	checkResApiGroup=""
	checkResNamespace="default"

	if allowed , reason , err:= k.CheckSelfRole( checkVerb, checkResName, checkSubResName , checkResInstanceName  , checkResApiGroup , checkResNamespace  ) ; err!=nil{
		fmt.Printf("error: %v \n" , err )
	}else {
		fmt.Printf("allowed?%v , reason: %s \n" ,allowed , reason )
	}


	//========================== resource instance

	checkVerb=k8s.VerbGet
	checkResName="configmaps"
	checkSubResName=""
	checkResInstanceName="tom-config"
	checkResApiGroup=""
	checkResNamespace="default"

	if allowed , reason , err:= k.CheckSelfRole( checkVerb, checkResName, checkSubResName , checkResInstanceName  , checkResApiGroup , checkResNamespace  ) ; err!=nil{
		fmt.Printf("error: %v \n" , err )
	}else {
		fmt.Printf("allowed?%v , reason: %s \n" ,allowed , reason )
	}



	//========================== noresource

	checkVerb=k8s.VerbGet
	checkResName="/health"

	if allowed , reason , err:= k.CheckSelfRole( checkVerb, checkResName, "" , ""  , "" , ""  ) ; err!=nil{
		fmt.Printf("error: %v \n" , err )
	}else {
		fmt.Printf("allowed?%v , reason: %s \n" ,allowed , reason )
	}



}


//==================================================


func Test_info_configmap(t *testing.T){

	k8s.EnableLog=false
	k:=k8s.K8sClient{}




	//------------ 当本地 cache  发生资源变化时，事件回调函数

	//注意，当创建informer后，本地的cache 就会开始从K8S 同步 并添加 现有的 资源到本地的 cache 
	// 所以，你会发现，当创建informer初始 ，HandlerAddFunc 回调就会被 调用多次
	AddEventChannel := make(chan *corev1.ConfigMap , 100 )
	HandlerAddFunc := func(obj interface{}) {
				// 转化成相应的资源
				// https://godoc.org/k8s.io/api/core/v1#ConfigMap
				instance := obj.(*corev1.ConfigMap )
				fmt.Printf(" new  instance evenvt : name=%v , nameSpace=%v  \n", 
						instance.ObjectMeta.Name  , instance.ObjectMeta.Namespace   )

				// 通过channel 同步到外部
				AddEventChannel <- instance
			}

	DeleteFunc := func(obj interface{}) {
				// 转化成相应的资源
				// https://godoc.org/k8s.io/api/core/v1#ConfigMap
				instance := obj.(*corev1.ConfigMap )
				fmt.Printf(" del  instance evenvt : name=%v , nameSpace=%v ,  data=%v \n", 
						instance.ObjectMeta.Name  , instance.ObjectMeta.Namespace  , instance.Data )
			}

	UpdateFunc := func( oldObj, newObj interface{})  {

				// 如果 informer使用了 sync ， 那么需要添加如下 检查代码，以过滤掉无用的事件
				newDepl := newObj.(*corev1.ConfigMap)
				oldDepl := oldObj.(*corev1.ConfigMap)
				if newDepl.ResourceVersion == oldDepl.ResourceVersion {
					// Periodic resync will send update events for all known Deployments.
					// Two different versions of the same Deployment will always have different RVs.
					return
				}

				fmt.Printf(" update  instance evenvt : name=%v  ; new data=%v \n" , 
						 newDepl.ObjectMeta.Name ,  newDepl.Data   )
			}

	EventHandlerFuncs:=&cache.ResourceEventHandlerFuncs{
		AddFunc: HandlerAddFunc , 
		UpdateFunc: UpdateFunc , 
		DeleteFunc: DeleteFunc ,
	}
	//EventHandlerFuncs:= (*cache.ResourceEventHandlerFuncs)nil

	// https://github.com/kubernetes/client-go/blob/be97aaa976ad58026e66cd9af5aaf1b006081f09/informers/generic.go#L87
	resType:= corev1.SchemeGroupVersion.WithResource("configmaps")

	// 注册 informer , 开始watch 全部 namespaces 下的 configmaps 信息
	// 注意， CreateInformer 调用后，各种回调 就会开始收到  存量configmap 的 各种事件！
	// 经过测试，就是 api server 各种重启， info 机制 也能 继续正常工作
	genericlister , stopWatchCh , e:=k.CreateInformer(resType , EventHandlerFuncs ) 
	if e!=nil {
		fmt.Printf(  "failed : %v " , e )
		t.FailNow()
	}
	// 关闭watch
	defer close(stopWatchCh) 


    fmt.Printf("------------------------- \n")
	time.Sleep(30*time.Second )
	//======= 通过 lister，配合 label selector ，  能获取当前最新的 指定资源 数据
     // 从 lister 中获取所有 items
     // https://godoc.org/k8s.io/client-go/tools/cache#GenericLister
     // https://github.com/kubernetes/client-go/blob/be97aaa976ad58026e66cd9af5aaf1b006081f09/tools/cache/listers.go#L112
     // List(selector labels.Selector) (ret []runtime.Object, err error)
     // https://github.com/kubernetes/client-go/tree/be97aaa976ad58026e66cd9af5aaf1b006081f09/listers
     // 可基于 lable 来选择性的返回 objects across namespaces
    instanceList, e2 := genericlister.List( labels.Everything() )
    if e2 != nil {
		fmt.Printf(  "failed : %v " , e2 )
		t.FailNow()
    }
    for n , item := range instanceList {
    	instance:=item.(*corev1.ConfigMap)
	    fmt.Printf("%d : %v \n ", n ,  instance.ObjectMeta.Name   )    	
    }


    fmt.Printf("------------------------- \n")
    // 通过Get方法，嫩头获取指定的 资源实例， name="nameSpace/instanceName"
    // https://github.com/kubernetes/client-go/blob/be97aaa976ad58026e66cd9af5aaf1b006081f09/tools/cache/listers.go#L116
    instance, _ := genericlister.Get( "kube-system/kube-proxy" )
	fmt.Printf(" %v \n ",  instance   )    	
    

    fmt.Printf("------------------------- \n")
    // 通过 ByNamespace 方法, 能获取指定的namespace下的 资源实例
    // https://github.com/kubernetes/client-go/blob/be97aaa976ad58026e66cd9af5aaf1b006081f09/tools/cache/listers.go#L118    
    instanceList1, e3 := genericlister.ByNamespace( "default" ).List( labels.Everything() )
    if e3 != nil {
		fmt.Printf(  "failed : %v " , e3 )
		t.FailNow()
    }
    for n , item := range instanceList1 {
    	instance:=item.(*corev1.ConfigMap)
	    fmt.Printf("%d : %v \n ", n ,  instance.ObjectMeta.Name   )    	
    }




}



func Test_info_node(t *testing.T){

	k8s.EnableLog=false
	k:=k8s.K8sClient{}


	//------------ 当本地 cache  发生资源变化时，事件回调函数

	//注意，当创建informer后，本地的cache 就会开始从K8S 同步 并添加 现有的 资源到本地的 cache 
	// 所以，你会发现，当创建informer初始 ，HandlerAddFunc 回调就会被 调用多次
	AddEventChannel := make(chan *corev1.Node , 100 )
	HandlerAddFunc := func(obj interface{}) {
				// 转化成相应的资源
				// https://godoc.org/k8s.io/api/core/v1#ConfigMap
				instance := obj.(*corev1.Node )
				fmt.Printf(" new  node  : name=%v   \n", 
						instance.ObjectMeta.Name     )

				// 通过channel 同步到外部
				AddEventChannel <- instance
			}

	DeleteFunc := func(obj interface{}) {
				// 转化成相应的资源
				// https://godoc.org/k8s.io/api/core/v1#ConfigMap
				instance := obj.(*corev1.Node )
				fmt.Printf(" del node evenvt : name=%v  \n", 
						instance.ObjectMeta.Name    )
			}

	UpdateFunc := func( oldObj, newObj interface{})  {

				// 如果 informer使用了 sync ， 那么需要添加如下 检查代码，以过滤掉无用的事件
				newDepl := newObj.(*corev1.Node)
				oldDepl := oldObj.(*corev1.Node)
				if newDepl.ResourceVersion == oldDepl.ResourceVersion {
					// Periodic resync will send update events for all known Deployments.
					// Two different versions of the same Deployment will always have different RVs.
					return
				}

				fmt.Printf(" update node evenvt : name=%v  \n" , 
						 newDepl.ObjectMeta.Name   )
			}

	EventHandlerFuncs:=&cache.ResourceEventHandlerFuncs{
		AddFunc: HandlerAddFunc , 
		UpdateFunc: UpdateFunc , 
		DeleteFunc: DeleteFunc ,
	}
	//EventHandlerFuncs:= (*cache.ResourceEventHandlerFuncs)nil

	// https://github.com/kubernetes/client-go/blob/be97aaa976ad58026e66cd9af5aaf1b006081f09/informers/generic.go#L182
	resType:= corev1.SchemeGroupVersion.WithResource("nodes")

	// 注册 informer , 开始watch 全部 namespaces 下的 configmaps 信息
	genericlister , stopWatchCh , e:=k.CreateInformer(resType , EventHandlerFuncs ) 
	if e!=nil {
		fmt.Printf(  "failed : %v " , e )
		t.FailNow()
	}
	// 关闭watch
	defer close(stopWatchCh) 


    fmt.Printf("------------------------- \n")
	time.Sleep(30*time.Second )
	//======= 通过 lister，配合 label selector ，  能获取当前最新的 指定资源 数据
    instanceList, e2 := genericlister.List( labels.Everything() )
    if e2 != nil {
		fmt.Printf(  "failed : %v " , e2 )
		t.FailNow()
    }
    for n , item := range instanceList {
    	instance:=item.(*corev1.Node)
	    fmt.Printf("%d : %v \n ", n ,  instance.ObjectMeta.Name   )    	
    }


}



//==================================================


func Test_configmap(t *testing.T){

	k8s.EnableLog=true

	
	k:=k8s.K8sClient{}



	namespace:="kube-system"
	name:="myconfig"
	// https://godoc.org/k8s.io/api/core/v1#ConfigMap
	configmapData:=&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: name , 
			Namespace: namespace , 
			Labels: map[string]string{},
			Annotations: map[string]string{},
		},
		Data: map[string]string {
			"key": "my value" ,
		},
	}



	//k.DeleteConfigmap( namespace ,  name ) 
	//fmt.Println(  "succeeded to delete configmap " )


	if  e:=k.ApplyConfigmap(  configmapData ) ; e!=nil {
		fmt.Printf(  "failed to update configmap : %v " , e )
		t.FailNow()
	}
	fmt.Println(  "succeeded to update configmap " )



	// if _ , e:=k.CreateConfigmap( namespace ,  configmapData ) ; e!=nil {
	// 	fmt.Printf(  "failed to create configmap : %v " , e )
	// 	t.FailNow()
	// }
	// fmt.Println(  "succeeded to create configmap " )




	if  cmDetailList , e:=k.ListConfigmap( namespace  ) ; e!=nil {
		fmt.Printf(  "failed to create configmap : %v " , e )
		t.FailNow()
	}else{
		fmt.Printf(  "list configmap data: %v  =======\n" , cmDetailList[namespace+"/"+name] )
	}



	if  cmdata , e:=k.GetConfigmap( namespace , name ) ; e!=nil {
		fmt.Printf(  "failed to create configmap : %v " , e )
		t.FailNow()
	}else{
		fmt.Printf(  "get configmap data: %v  =========\n" , cmdata )
	}





}



func Test_ns(t *testing.T){

	k8s.EnableLog=true
	k:=k8s.K8sClient{}

	if nsList , _ , e:=k.GetNamespace() ; e!=nil {
		fmt.Printf(  "failed to GetNamespace  : %v " , e )
		t.FailNow()

	}else{

		fmt.Printf("all namesapces: %v \n" , nsList )
	}

}


//----------------------


func testLeader( id string){
	k8s.EnableLog=true
	k:=k8s.K8sClient{}

	// lease 名字必须是小写的，由数字，字母，'-' or '.' 等组成, 其必须是 以字母来开头和结尾
	// a lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters, '-' or '.',
	// and must start and end with an alphanumeric character
	// (e.g. 'example.com', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')
	leaseLockName:="welanlease"

	leaseLockNamespace:="default"
	// 注意：无论几个候选人上来，用什么样的ip，只要是 myId 相同， 他们都会拿到leader ， 简单说，只认 myId
	myId:=id
	//如果持续 leaseDuration 没有 续租，则会丢失leader
	leaseDuration:=uint(20)
	// renewDeadline 获取leader 后， 自动续租的周期
	renewDeadline:=uint(5)
	// 尝试获取 leader 的间隔
	retryLockPeriod:=uint(2)
	//newLeaderHandler:=nil
	newLeaderHandler:=func(identity string){
		fmt.Printf(" leader change event: new leader %v for lease %v/%v \n" , identity , leaseLockNamespace ,leaseLockName  )
	}

	acquireLeaderChan , lostLeaderChan , cancelLease,  e:=k.Lease( leaseLockName , leaseLockNamespace , myId ,
		leaseDuration , renewDeadline , retryLockPeriod , newLeaderHandler )
	if e!=nil {
		fmt.Printf("error, failed to create lease for %v/%v , reason=%v \n",
			leaseLockNamespace , leaseLockName , e)
	}
	defer func(){
		cancelLease()
	}()

	fmt.Printf("%v waiting for the leader of %v/%v \n", myId , leaseLockNamespace ,leaseLockName  )
	<-acquireLeaderChan

	fmt.Printf(" I (%v) am leader for %v/%v \n", myId , leaseLockNamespace ,leaseLockName  )
	// do business
	for  i:=0 ; i<20 ; i++  {
		select{
		case <-lostLeaderChan:
			fmt.Printf(" error, I(%v) lost leader for %v/%v \n",myId ,  leaseLockNamespace ,leaseLockName  )
		case <-time.After(2*time.Second):
		}
	}

	// cancel my leader
	cancelLease()
	fmt.Printf("%v quit leader for %v/%v \n", myId , leaseLockNamespace ,leaseLockName  )

	time.Sleep(2*time.Second)
}

func Test_leaseTom(t *testing.T) {
	testLeader("tom")
}
func Test_leaseJim(t *testing.T) {
	testLeader("jim")
}

func Test_leaseBoth(t *testing.T) {
	go testLeader("tom")
	go testLeader("jim")
	time.Sleep(120*time.Second)
}





