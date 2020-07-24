package golib_k8s
import (
	"os"
	"fmt"
	"path/filepath"
	goRuntime "runtime"
	"strings"
	"strconv"
	"context"
	"time"

	"k8s.io/client-go/rest"	
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"	
	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/retry"


	authorizationClientv1 "k8s.io/client-go/kubernetes/typed/authorization/v1"
	authorizationv1 "k8s.io/api/authorization/v1"

	kubeinformers "k8s.io/client-go/informers"
	cache "k8s.io/client-go/tools/cache"
	//"k8s.io/apimachinery/pkg/labels"


)

//=====================================================
/*
github 
https://github.com/kubernetes/client-go
godoc
https://godoc.org/k8s.io/client-go/kubernetes

重要的库
https://godoc.org/k8s.io/api/core/v1  关于容器、yam , configmap 的一些基本结构
https://godoc.org/k8s.io/api/apps/v1  APIGROUP apps 的一些结构，如 daemonset , deployment , replicaset 等资源

https://godoc.org/k8s.io/client-go/informers



!!!!!!!!!!!!!!!!!!!!!!!!
example: https://github.com/kubernetes/client-go/tree/master/examples
!!!!!!!!!!!!!!!!!!!!!!!!





//--------------------- pods

func (c *K8sClient)ListPods( namespace string ) (  podDetailList map[string]corev1.Pod , e error )

func (c *K8sClient)CheckPodHealthy( namespace string , podName string ) (  exist bool , e error )

//--------------------- configmap

func (c *K8sClient)ListConfigmap( namespace string ) (  cmDetailList map[string] corev1.ConfigMap , e error )
//如果configmap不存在，error 存在
func (c *K8sClient)GetConfigmap( namespace , name string ) (  cmData *corev1.ConfigMap , e error )
func (c *K8sClient)DeleteConfigmap( namespace , name string  ) ( e error )
//如果存在，则更新，如果不存在，则创建它
func (c *K8sClient)ApplyConfigmap( cmData *corev1.ConfigMap ) ( e error )


//--------------------- deployment
#使用传统的方式，deployment 的数据结构定义固定，虽然 不和 版本耦合
func (c *K8sClient)GetDeploymentTyped( namespace string ,  deploymentName string ) (  deploymentBasicInfo []map[string]string , deploymentDetailList  []appsv1.Deployment , e error )

#使用灵活的方式，使用 灵活的 unstructured.Unstructured 结构体来代表  deployment的数据结构
func (c *K8sClient)GetDeployment( namespace string ,  deploymentName string ) (  deploymentBasicInfo []map[string]string , deploymentDetailList  []unstructured.Unstructured , e error )

func (c *K8sClient)CreateDeploymentTyped( namespace string , deploymentSpec *appsv1.Deployment ) ( e error )

func (c *K8sClient)CreateDeployment( namespace string , deploymentYaml *unstructured.Unstructured ) ( e error )

func (c *K8sClient)DelDeploymentTyped( namespace string , deploymentName string ) ( e error )

func (c *K8sClient)DelDeployment( namespace string , deploymentName string ) ( e error )

func (c *K8sClient)UpdateDeploymentTyped( namespace string , deploymentName string , handler func(*appsv1.Deployment)error  ) ( e error )

func (c *K8sClient)UpdateDeployment( namespace string , deploymentName string , handler func(*unstructured.Unstructured)error  ) ( e error )


//----- RBAC
func (c *K8sClient)CheckUserRole( userName string  , userGroupName []string , checkVerb VerbType ,	checkResName string, checkSubResName string ,checkResInstanceName string , checkResApiGroup string , checkResNamespace string ) ( allowed bool , reason string , e error )

func (c *K8sClient)CheckSelfRole(  checkVerb VerbType ,	checkResName string, checkSubResName string ,checkResInstanceName string , checkResApiGroup string , checkResNamespace string ) ( allowed bool , reason string , e error )

// informer
func (c *K8sClient)CreateInformer(  resourceType  schema.GroupVersionResource ,  EventHandlerFuncs *cache.ResourceEventHandlerFuncs )  ( lister cache.GenericLister , stopWatchCh chan struct{} ,  e error ) {



*/


//=======================================================

var (
	//for log
    EnableLog=false
    // for config
    ScInPodPath="/var/run/secrets/kubernetes.io/serviceaccount"
    KubeConfigPath=filepath.Join(os.Getenv("HOME"), ".kube", "config")
    RequestTimeOut=time.Duration(10)

)

type K8sClient struct {
	//Client *kubernetes.Clientset
	Config *rest.Config
}


//============tools==========


func getFileName( path string ) string {
    b:=strings.LastIndex(path,"/")
    if b>=0 {
        return path[b+1:]
    }else{
        return path
    }
}

func log( format string, a ...interface{} ) (n int, err error) {

    if EnableLog {

		prefix := ""
	    funcName,filepath ,line,ok := goRuntime.Caller(1)
	    if ok {
	    	file:=getFileName(filepath)
	    	funcname:=getFileName(goRuntime.FuncForPC(funcName).Name())
	    	prefix += "[" + file + " " + funcname + " " + strconv.Itoa(line) +  "]     "
	    }

        return fmt.Printf(prefix+format , a... )    
    }
    return  0,nil
}



func existFile( filePath string ) bool {
	if info , err := os.Stat(filePath) ; err==nil {
		if ! info.IsDir() {
			return true
		}
    }
    return false
}


func ExistDir( dirPath string ) bool {
	if info , err := os.Stat(dirPath) ; err==nil {
		if info.IsDir() {
			return true
		}
    }
    return false
}



//===========================

func (c *K8sClient) autoConfig( ) error  {

	// 也可使用如下简单的代码实现
	// import "k8s.io/client-go/tools/clientcmd"
	// masterUrl:=""
	// kubeconfigPath:=""
	// // If neither masterUrl or kubeconfigPath are passed in we fallback to inClusterConfig. 
	// // If inClusterConfig fails, we fallback to the default config
	// cfg, e1 := clientcmd.BuildConfigFromFlags(masterUrl , kubeconfigPath )
 // 	if e1 != nil {
	// 	return e1
 // 	}

	// c.Config=cfg
	// return nil


	var config *rest.Config
	var err error 

	if existFile(KubeConfigPath)==true {
		log("Outside of pod , try to get config from kube config file \n")

		config, err = clientcmd.BuildConfigFromFlags("", KubeConfigPath)
		if err != nil {
			return fmt.Errorf("failed to get config from kube config=%v , info=%v" , KubeConfigPath , err )
		}

	}else if ExistDir(ScInPodPath)==true {
		log("In pod , try to get config from serviceaccount \n")

		config, err = rest.InClusterConfig()
		if err != nil {
			return fmt.Errorf("failed to get config from serviceaccount=%v , info=%v" , ScInPodPath , err )
		}


	}else{
		return fmt.Errorf("failed to get config " )
	}

	log("%v \n" , config )

	c.Config=config

	return nil
}


//=============namespace=======================


//=============node=======================
/*
output:
	nodeList [{"Name":..}]
	nodeDetailList []Node  // struct defination: https://godoc.org/k8s.io/api/core/v1#Node
*/
func (c *K8sClient)GetNodes(  ) (  nodeList []map[string]string , nodeDetailList []corev1.Node , e error ){

	if c.Config == nil {
		if e1:=c.autoConfig() ; e1 !=nil {
			e=fmt.Errorf("failed to config : %v " , e )
			return 
		}
	}

	client, e1 := kubernetes.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 

	// https://godoc.org/k8s.io/client-go/kubernetes/typed/core/v1#CoreV1Client.Nodes
	// https://github.com/kubernetes/client-go/blob/master/kubernetes/typed/core/v1/node.go#L40
	info , err := client.CoreV1().Nodes().List( ctx ,  metav1.ListOptions{})
	if err != nil {
		e=fmt.Errorf("%v" , err )
		return
	}
	if info==nil {
		return
	}

	for _ , v := range info.Items {

		internalIp:=""

		for _ , m := range v.Status.Addresses {
			if m.Type == corev1.NodeInternalIP {
				internalIp=m.Address
				break
			}
		}

		nodeList=append( nodeList , map[string]string {
			"Name": v.ObjectMeta.Name ,
			"NodeIp": internalIp ,
		})
	}

	nodeDetailList=info.Items

	return
}

//=============pod=======================
/*
input:
	namespace="" , get all namespaces
output:
	podDetailList map[string]*corev1.Pod  // struct defination: https://godoc.org/k8s.io/api/core/v1#Pod
*/
func (c *K8sClient)ListPods( namespace string ) (  podDetailList map[string]corev1.Pod , e error ){

	if c.Config == nil {
		if e1:=c.autoConfig() ; e1 !=nil {
			e=fmt.Errorf("failed to config : %v " , e )
			return 
		}
	}

	if len(namespace)==0 {
		log("get pods from all namespaces \n")
	}else{
		log("get pods from namespace=%v \n" , namespace )
	}

	client, e1 := kubernetes.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 

	// https://godoc.org/k8s.io/client-go/kubernetes/typed/core/v1#PodInterface
	// https://github.com/kubernetes/client-go/blob/master/kubernetes/typed/core/v1/pod.go#L40
	pods, err := client.CoreV1().Pods(namespace).List( ctx , metav1.ListOptions{})
	if err != nil {
		e=fmt.Errorf("%v" , err )
		return
	}
	if pods==nil {
		return
	}


	log("got pod num=%v \n" , len( pods.Items ) )
	log("TypeMeta=%v \n" ,  pods.TypeMeta  )
	log("ListMeta=%v \n" ,  pods.ListMeta  )

	podDetailList=map[string]corev1.Pod {}
	for _, k :=range pods.Items {
		podDetailList[ k.ObjectMeta.Namespace +"/"+k.ObjectMeta.Name]=k
	}

	return 
}


/*
input:
	namespace
*/
func (c *K8sClient)CheckPodHealthy( namespace string , podName string ) (  exist bool , e error ){

	exist=false

	if c.Config == nil {
		if e1:=c.autoConfig() ; e1 !=nil {
			e=fmt.Errorf("failed to config : %v " , e )
			return 
		}
	}



	if len(namespace)==0 {
		e=fmt.Errorf("empty namespace " )
		return
	}
	if len(podName)==0 {
		e=fmt.Errorf("empty podName " )
		return
	}
	log("get pod=%v from namespace=%v \n" , podName , namespace )
	

	Client, e1 := kubernetes.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 

	// https://github.com/kubernetes/client-go/blob/master/kubernetes/typed/core/v1/pod.go#L71
	_, err := Client.CoreV1().Pods(namespace).Get( ctx , podName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		e=fmt.Errorf( "Pod %s in namespace %s not found\n", podName, namespace )
		return

	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		e=fmt.Errorf("Error pod status : %v\n", statusError.ErrStatus.Message)
		return

	} else if err != nil {
		e=fmt.Errorf( "%v", err )
		return

	}else{
		exist=true
	}


	return 
}


//============= configmap =======================

/*
input:
	namespace="" , get all namespaces
output:
	cmList [{"Name":.. "Namespace":.. "UID":...}]
	cmDetailList map[string]corev1.ConfigMap  // index=Namespace+"/"+Name
*/
func (c *K8sClient)ListConfigmap( namespace string ) (  cmDetailList map[string] corev1.ConfigMap , e error ){


	if c.Config == nil {
		if e1:=c.autoConfig() ; e1 !=nil {
			e=fmt.Errorf("failed to config : %v " , e )
			return 
		}
	}


	if len(namespace)==0 {
		log("get configmap from all namespaces \n")
	}else{
		log("get configmap from namespace=%v \n" , namespace )
	}

	client, e1 := kubernetes.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 

	// https://godoc.org/k8s.io/client-go/kubernetes/typed/core/v1#ConfigMapInterface
	// https://github.com/kubernetes/client-go/blob/master/kubernetes/typed/core/v1/configmap.go#L40
	cms, err := client.CoreV1().ConfigMaps(namespace).List( ctx , metav1.ListOptions{})
	if err != nil {
		e=fmt.Errorf("%v" , err )
		return
	}
	if cms==nil {
		return
	}


	log("got configmap num=%v \n" , len( cms.Items ) )
	log("TypeMeta=%v \n" ,  cms.TypeMeta  )
	log("ListMeta=%v \n" ,  cms.ListMeta  )

	cmDetailList=map[string] corev1.ConfigMap {}

	for _, k :=range cms.Items {
		cmDetailList[k.ObjectMeta.Namespace+"/"+k.ObjectMeta.Name]= k

	}

	return 
}



/*
input:
	namespace="" , get all namespaces
output:
	cmData *corev1.ConfigMap

如果configmap不存在，error 存在
*/
func (c *K8sClient)GetConfigmap( namespace , name string ) (  cmData *corev1.ConfigMap , e error ){

	if c.Config == nil {
		if e1:=c.autoConfig() ; e1 !=nil {
			e=fmt.Errorf("failed to config : %v " , e )
			return 
		}
	}


	if len(namespace)==0 {
		log("get configmap from all namespaces \n")
	}else{
		log("get configmap from namespace=%v \n" , namespace )
	}

	client, e1 := kubernetes.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 

	// https://godoc.org/k8s.io/client-go/kubernetes/typed/core/v1#ConfigMapInterface
	// https://github.com/kubernetes/client-go/blob/master/kubernetes/typed/core/v1/configmap.go#L40
	cmData, e = client.CoreV1().ConfigMaps(namespace).Get( ctx , name , metav1.GetOptions{})


	return 
}


//如果存在，则更新，如果不存在，则创建它
func (c *K8sClient)ApplyConfigmap( cmData *corev1.ConfigMap ) ( e error ){


	if c.Config == nil {
		if e1:=c.autoConfig() ; e1 !=nil {
			e=fmt.Errorf("failed to config : %v " , e )
			return 
		}
	}

	if len(cmData.ObjectMeta.Namespace)==0 {
		e=fmt.Errorf("miss Namespace" )
		return
	}



	client, e1 := kubernetes.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 

	// https://godoc.org/k8s.io/client-go/kubernetes/typed/core/v1#ConfigMapInterface
	// https://github.com/kubernetes/client-go/blob/master/kubernetes/typed/core/v1/configmap.go#L40
	// if existed {
	// 	log(" configmap %v exists , try to update it \n" , cmData.ObjectMeta.Name  )
	// 	_, e = client.CoreV1().ConfigMaps(cmData.ObjectMeta.Namespace).Update( ctx , cmData , metav1.UpdateOptions{})

	// }else{
	// 	log(" configmap %v miss , try to create it \n" , cmData.ObjectMeta.Name  )
	// 	_ ,  e = client.CoreV1().ConfigMaps(cmData.ObjectMeta.Namespace).Create( ctx , cmData , metav1.CreateOptions{})

	// }

	if _ , e1:= client.CoreV1().ConfigMaps(cmData.ObjectMeta.Namespace).Create( ctx , cmData , metav1.CreateOptions{}) ; e1!=nil {
		ctx, _ = context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 
		if _ , e2:=client.CoreV1().ConfigMaps(cmData.ObjectMeta.Namespace).Update( ctx , cmData , metav1.UpdateOptions{}) ; e2!=nil {
			e=fmt.Errorf("failed to create info=%v ; failed to update config=%v " , e1 , e2 )
		}

	}


	return 
}




func (c *K8sClient)DeleteConfigmap( namespace , name string  ) ( e error ){

	if c.Config == nil {
		if e1:=c.autoConfig() ; e1 !=nil {
			e=fmt.Errorf("failed to config : %v " , e )
			return 
		}
	}


	if len(namespace)==0 {
		e=fmt.Errorf("empty namespace "  )
		return
	}
	if len(name)==0 {
		e=fmt.Errorf("empty configmap name "  )
		return
	}
	log("delete configmap=%v under namespace=%v \n" ,  name , namespace  )


	client, e1 := kubernetes.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 

	// https://godoc.org/k8s.io/client-go/kubernetes/typed/core/v1#ConfigMapInterface
	// https://github.com/kubernetes/client-go/blob/master/kubernetes/typed/core/v1/configmap.go#L40
	err := client.CoreV1().ConfigMaps(namespace).Delete( ctx , name , metav1.DeleteOptions{})
	if err != nil {
		e=fmt.Errorf("%v" , err )
		return
	}

	return 

}






// func (c *K8sClient)CreateConfigmap( namespace string ,  configmapData *corev1.ConfigMap ) ( result *corev1.ConfigMap , e error ){

// 	if c.Config == nil {
// 		if e1:=c.autoConfig() ; e1 !=nil {
// 			e=fmt.Errorf("failed to config : %v " , e )
// 			return 
// 		}
// 	}


// 	if len(namespace)==0 {
// 		e=fmt.Errorf("empty namespace "  )
// 		return
// 	}
// 	if configmapData==nil {
// 		e=fmt.Errorf("empty configmap name "  )
// 		return
// 	}
// 	log("create configmap=%v under namespace=%v \n" ,  configmapData.ObjectMeta.Name , namespace  )


// 	client, e1 := kubernetes.NewForConfig(c.Config)
// 	if e1 != nil {
// 		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
// 		return
// 	}

// 	ctx, _ := context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 

// 	// https://godoc.org/k8s.io/client-go/kubernetes/typed/core/v1#ConfigMapInterface
// 	// https://github.com/kubernetes/client-go/blob/master/kubernetes/typed/core/v1/configmap.go#L40
// 	result ,  e = client.CoreV1().ConfigMaps(namespace).Create( ctx , configmapData , metav1.CreateOptions{})


// 	return 

// }





//=============deployment=======================


/*
input:
	namespace    // namespace=""  for all namespace
output:
	deploymentDetailInfo  []appsv1.Deployment  // // type Deployment : https://godoc.org/k8s.io/api/apps/v1#Deployment
*/
func (c *K8sClient)ListDeploymentTyped( namespace string  ) (   deploymentDetailList  map[string]*appsv1.Deployment , e error ){

	if c.Config == nil {
		if e1:=c.autoConfig() ; e1 !=nil {
			e=fmt.Errorf("failed to config : %v " , e )
			return 
		}
	}





	Client, e1 := kubernetes.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}

		if len(namespace)==0 {
			log("get all deployment from all namesapces \n")
		}else{
			log("get all deployment from namesapces=%s \n" , namespace )
		}

		ctx, _ := context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 

		// https://godoc.org/k8s.io/client-go/kubernetes/typed/apps/v1#DeploymentInterface 
		// https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#ListOptions
		result, err  := Client.AppsV1().Deployments(namespace).List( ctx ,  metav1.ListOptions{} )
		if err != nil {
			e=fmt.Errorf( "%v", err )
			return
		}

		//type DeploymentList:  https://godoc.org/k8s.io/api/apps/v1#DeploymentList
		log("TypeMeta=%v \n" , result.TypeMeta )
		log("ListMeta=%v \n" , result.ListMeta )
		log("Deployment number=%v \n", len(result.Items)  )

		deploymentDetailList = map[string]*appsv1.Deployment {}
		for _, v :=range result.Items {
			// type Deployment : https://godoc.org/k8s.io/api/apps/v1#Deployment
			deploymentDetailList[v.ObjectMeta.Namespace+"/"+v.ObjectMeta.Name]=&v
		}


	log("succeeded  \n"  )

	return 
}


/*
input:
	namespace  // namespace="" for all namespace
	deploymentName  // set when namespace!=""
output:
	deploymentDetailList  // unstructured.Unstructured : https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured#Unstructured
*/

func (c *K8sClient)ListDeployment( namespace string   ) (   deploymentDetailList  map[string]*unstructured.Unstructured , e error ){

	if c.Config == nil {
		if e1:=c.autoConfig() ; e1 !=nil {
			e=fmt.Errorf("failed to config : %v " , e )
			return 
		}
	}



	// https://godoc.org/k8s.io/client-go/dynamic
	Client, e1 := dynamic.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}

	// https://godoc.org/k8s.io/apimachinery/pkg/runtime/schema
	// https://godoc.org/k8s.io/apimachinery/pkg/runtime/schema#GroupVersionResource
	deploymentRes := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}

		if len(namespace)==0 {
			log("get all deployment from all namesapces \n")
		}else{
			log("get all deployment from namesapces=%s \n" , namespace )
		}

		ctx, _ := context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 


		// https://godoc.org/k8s.io/api/apps/v1#Resource
		// https://godoc.org/k8s.io/client-go/dynamic#Interface
		// https://godoc.org/k8s.io/client-go/dynamic#NamespaceableResourceInterface
		list, err := Client.Resource(deploymentRes).Namespace(namespace).List( ctx , metav1.ListOptions{})
		if err != nil {
			e=fmt.Errorf(" info=%v  " , err  )
			return
		}

		//for list struct : https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured#UnstructuredList
		// log("%v \n" , list )
		log("%v \n" , list.Object )

		deploymentDetailList = map[string]*unstructured.Unstructured {}
		for _, d := range list.Items {
			//for list.Items struct:  https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured#Unstructured

				//get deployment name
				name:=d.GetName()
				//get GetNamespace name
				namespace:=d.GetNamespace()
				//get GetNamespace name
				//uid:=d.GetUID()

				deploymentDetailList[namespace+"/"+name]=&d

		}



	log("succeeded  \n"  )
	return 
}






/*
input:
	namespace    
output:
	deploymentDetailInfo  []appsv1.Deployment  // // type Deployment : https://godoc.org/k8s.io/api/apps/v1#Deployment
*/
func (c *K8sClient)CreateDeploymentTyped( namespace string , deploymentSpec *appsv1.Deployment ) ( e error ){

	if c.Config == nil {
		if e1:=c.autoConfig() ; e1 !=nil {
			e=fmt.Errorf("failed to config : %v " , e )
			return 
		}
	}



	if len(namespace)==0 {
		e=fmt.Errorf("empty namespace " )
		return
	}


	Client, e1 := kubernetes.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 

	// https://github.com/kubernetes/client-go/blob/master/kubernetes/typed/apps/v1/deployment.go#L117
	//type DeploymentInterface:  https://godoc.org/k8s.io/client-go/kubernetes/typed/apps/v1#DeploymentInterface
	info , err := Client.AppsV1().Deployments(namespace).Create(ctx, deploymentSpec , metav1.CreateOptions{} ) 
	if err != nil {
		e=fmt.Errorf( "%v", err )
		return
	}

	log("succeeded  to create deployment: %v \n" , info )

	return 
}




func (c *K8sClient)CreateDeployment( namespace string , deploymentYaml *unstructured.Unstructured ) ( e error ){

	if c.Config == nil {
		if e1:=c.autoConfig() ; e1 !=nil {
			e=fmt.Errorf("failed to config : %v " , e )
			return 
		}
	}


	if len(namespace)==0 {
		e=fmt.Errorf("empty namespace " )
		return
	}
	if deploymentYaml==nil {
		e=fmt.Errorf("empty yaml " )
		return
	}

	// https://godoc.org/k8s.io/client-go/dynamic
	Client, e1 := dynamic.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 


	// https://godoc.org/k8s.io/apimachinery/pkg/runtime/schema
	// https://godoc.org/k8s.io/apimachinery/pkg/runtime/schema#GroupVersionResource
	deploymentRes := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	// https://godoc.org/k8s.io/api/apps/v1#Resource
	// https://godoc.org/k8s.io/client-go/dynamic#NamespaceableResourceInterface
	result, err := Client.Resource(deploymentRes).Namespace(namespace).Create(ctx , deploymentYaml, metav1.CreateOptions{})
	if err != nil {
		e=fmt.Errorf(" info=%v  " , err  )
		return
	}

	log("succeeded to create %v \n" , result.GetName() )
	return 
}




/*
input:
	namespace    // namespace=""  for all namespace
output:
	deploymentDetailInfo  []appsv1.Deployment  // // type Deployment : https://godoc.org/k8s.io/api/apps/v1#Deployment
*/
func (c *K8sClient)DelDeploymentTyped( namespace string , deploymentName string ) ( e error ){

	if c.Config == nil {
		if e1:=c.autoConfig() ; e1 !=nil {
			e=fmt.Errorf("failed to config : %v " , e )
			return 
		}
	}



	Client, e1 := kubernetes.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}


	// https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#DeletionPropagation
	deletePolicy := metav1.DeletePropagationForeground

	ctx, _ := context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 

	//type DeploymentInterface:  https://godoc.org/k8s.io/client-go/kubernetes/typed/apps/v1#DeploymentInterface
	// https://github.com/kubernetes/client-go/blob/master/kubernetes/typed/apps/v1/deployment.go#L160
	err := Client.AppsV1().Deployments(namespace).Delete(ctx , deploymentName , 
			// https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#DeleteOptions
			metav1.DeleteOptions{
				PropagationPolicy: &deletePolicy    })
	if err != nil {
		e=fmt.Errorf( "%v", err )
		return
	}

	log("succeeded  \n"  )

	return 
}




func (c *K8sClient)DelDeployment( namespace string , deploymentName string ) ( e error ){

	if c.Config == nil {
		if e1:=c.autoConfig() ; e1 !=nil {
			e=fmt.Errorf("failed to config : %v " , e )
			return 
		}
	}


	if len(namespace)==0 {
		e=fmt.Errorf("empty namespace " )
		return
	}
	if len(deploymentName)==0 {
		e=fmt.Errorf("empty deploymentName " )
		return
	}

	// https://godoc.org/k8s.io/client-go/dynamic
	Client, e1 := dynamic.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}

	// https://godoc.org/k8s.io/apimachinery/pkg/runtime/schema
	// https://godoc.org/k8s.io/apimachinery/pkg/runtime/schema#GroupVersionResource
	deploymentRes := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	// https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#DeletionPropagation
	deletePolicy := metav1.DeletePropagationForeground
	// https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#DeleteOptions
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}

	ctx, _ := context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 

	// https://godoc.org/k8s.io/api/apps/v1#Resource
	// https://godoc.org/k8s.io/client-go/dynamic#NamespaceableResourceInterface	
	if err := Client.Resource(deploymentRes).Namespace(namespace).Delete(ctx , deploymentName, deleteOptions); err != nil {
		e=fmt.Errorf(" info=%v  " , err  )
		return
	}

	log("succeeded to delete %v \n" , deploymentName )
	return 
}




func (c *K8sClient)UpdateDeploymentTyped( namespace string , deploymentName string , handler func(*appsv1.Deployment)error  ) ( e error ){

	if c.Config == nil {
		if e1:=c.autoConfig() ; e1 !=nil {
			e=fmt.Errorf("failed to config : %v " , e )
			return 
		}
	}


	if len(namespace)==0 {
		e=fmt.Errorf("empty namespace " )
		return
	}
	if len(deploymentName)==0 {
		e=fmt.Errorf("empty namespace " )
		return
	}
	if handler==nil {
		e=fmt.Errorf("empty yaml " )
		return
	}

	// https://godoc.org/k8s.io/client-go/dynamic
	Client, e1 := kubernetes.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}
	log("begin to update deployment %v under namespace %v \n" , deploymentName , namespace )

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Retrieve the latest version of Deployment before attempting update
		// RetryOnConflict uses exponential backoff to avoid exhausting the apiserver
		deploymentDetailList , e:=c.ListDeploymentTyped( namespace  ) 
		if e!=nil {
			return fmt.Errorf( "failed to get deploymen=%v , info=%v " , e )
		}
		result:=deploymentDetailList[namespace+"/"+deploymentName]
		log("got deployment %v yaml under namespace %v  \n" , deploymentName  , namespace )

		//log("%v\n" , result )
		if err:=handler(result) ; err!=nil {
			return fmt.Errorf( "failed to call hander info=%v " , err )
		}
		//log("%v\n" , result )

		ctx, _ := context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 

		//type DeploymentInterface:  https://godoc.org/k8s.io/client-go/kubernetes/typed/apps/v1#DeploymentInterface
		// https://github.com/kubernetes/client-go/blob/master/kubernetes/typed/apps/v1/deployment.go#L130
		_, err := Client.AppsV1().Deployments(namespace).Update(ctx , result , metav1.UpdateOptions{} )
		if err != nil {
			return fmt.Errorf( "failed to update deploymen=%v , info=%v " , deploymentName , err )
		}
		return nil
	})

	if retryErr != nil {
		e=retryErr
		return
	}

	log("succeeded to update deployment %v under namespace %v \n" , deploymentName , namespace )
	return 

}



func (c *K8sClient)UpdateDeployment( namespace string , deploymentName string , handler func(*unstructured.Unstructured)error  ) ( e error ){

	if c.Config == nil {
		if e1:=c.autoConfig() ; e1 !=nil {
			e=fmt.Errorf("failed to config : %v " , e )
			return 
		}
	}


	if len(namespace)==0 {
		e=fmt.Errorf("empty namespace " )
		return
	}
	if len(deploymentName)==0 {
		e=fmt.Errorf("empty namespace " )
		return
	}
	if handler==nil {
		e=fmt.Errorf("empty yaml " )
		return
	}

	// https://godoc.org/k8s.io/client-go/dynamic
	Client, e1 := dynamic.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}


	log("begin to update deployment %v under namespace %v \n" , deploymentName , namespace )

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Retrieve the latest version of Deployment before attempting update
		// RetryOnConflict uses exponential backoff to avoid exhausting the apiserver
		deploymentDetailList , e:=c.ListDeployment( namespace  ) 
		if e!=nil {
			return fmt.Errorf( "failed to get deploymen=%v , info=%v " , e )
		}
		result:=deploymentDetailList[namespace+"/"+deploymentName]
		log("got deployment %v yaml under namespace %v  \n" , deploymentName  , namespace )

		//log("%v\n" , result )
		if err:=handler(result) ; err!=nil {
			return fmt.Errorf( "failed to call hander info=%v " , err )
		}
		//log("%v\n" , result )


		ctx, _ := context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 

		// https://godoc.org/k8s.io/apimachinery/pkg/runtime/schema
		// https://godoc.org/k8s.io/apimachinery/pkg/runtime/schema#GroupVersionResource
		deploymentRes := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
		// https://godoc.org/k8s.io/api/apps/v1#Resource
		// https://godoc.org/k8s.io/client-go/dynamic#NamespaceableResourceInterface
		_, err := Client.Resource(deploymentRes).Namespace(namespace).Update(ctx , result, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf( "failed to update deploymen=%v , info=%v " , deploymentName , err )
		}
		return nil
	})

	if retryErr != nil {
		e=retryErr
		return
	}

	log("succeeded to update deployment %v under namespace %v \n" , deploymentName , namespace )
	return 

}








//=========== RBAC, check whether a user has the specified role ========================== 
 
/*
 write from example :  
	https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/auth/cani.go#L235-L260
	https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/auth/cani_test.go
*/




type VerbType string
const (
	VerbNone=VerbType("")
	// resource
	VerbGet=VerbType("get")
	VerbList=VerbType("list")
	VerbWatch=VerbType("watch")
	VerbCreate=VerbType("create")
	VerbUpdate=VerbType("update")
	VerbPatch=VerbType("patch")
	VerbDelete=VerbType("delete")
	VerbDeletecollection=VerbType("deletecollection")

	//nonSource
	VerbNoneGet=VerbType("get")
	VerbNonePost=VerbType("post")
	VerbNonePut=VerbType("put")
	VerbNonePatch=VerbType("patch")
	VerbNoneDelete=VerbType("delete")
	VerbNoneHead=VerbType("head")
	VerbNoneOptions=VerbType("options")


	VerbAll=VerbType("*")
)





/*
input:
checkNamespace
    // Namespace is the namespace of the action being requested.  Currently, there is no distinction between no namespace and all namespaces
    // "" (empty) is defaulted for LocalSubjectAccessReviews
    // "" (empty) is empty for cluster-scoped resources
    // "" (empty) means "all" for namespace scoped resources from a SubjectAccessReview or SelfSubjectAccessReview
    // +optional
checkResName:
	// 可为 K8S资源类的名字 可使用 kubectl api-resources 查询
	// 也可为 非K8S资源类的名字（自定义）
checkResApiGroup:
    // Group is the API Group of the Resource.  "*" means all.
    // +optional
    // 可使用 kubectl api-resources 查询每个 资源类的  api group

*/
func (c *K8sClient)CheckUserRole( userName string  , userGroupName []string , checkVerb VerbType ,	checkResName string, checkSubResName string ,checkResInstanceName string , checkResApiGroup string , checkResNamespace string ) ( allowed bool , reason string , e error ){

	e=nil
	allowed=false
	reason=""

	if checkVerb==VerbNone {
		e=fmt.Errorf("verb is none " )
		return
	}

	if c.Config == nil {
		if e1:=c.autoConfig() ; e1 !=nil {
			e=fmt.Errorf("failed to config : %v " , e )
			return 
		}
	}

	if len(userName)==0 && len(userGroupName)==0 {
		e=fmt.Errorf("empty userName and userGroupName " )
		return
	}
	if len(checkResName)==0 {
		e=fmt.Errorf("empty checkResName " )
		return
	}


	//  https://godoc.org/k8s.io/client-go/kubernetes/typed/authorization/v1#NewForConfig
	client, e1 := authorizationClientv1.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}

	var sar authorizationv1.SubjectAccessReview

	if strings.HasPrefix(checkResName, "/")==false {
		// ResourceURL
		log("check for Resource %s \n", checkResName )

		sar = authorizationv1.SubjectAccessReview{  // https://godoc.org/k8s.io/api/authorization/v1#SubjectAccessReview
			Spec: authorizationv1.SubjectAccessReviewSpec{   // https://godoc.org/k8s.io/api/authorization/v1#SubjectAccessReviewSpec
				User: userName ,  // check for a user
				Groups: userGroupName , // check for a user group
				//UID: "" ,
				ResourceAttributes: &authorizationv1.ResourceAttributes{  // https://godoc.org/k8s.io/api/authorization/v1#SubjectAccessReviewSpec
					Namespace:   checkResNamespace ,
					Verb:        string(checkVerb) ,
					Group:       checkResApiGroup , //"extensions",  // resource api group 
					Version:	"" , //  version of resource api group
									    // Version is the API Version of the Resource.  "*" means all.
									    // +optional
					Resource:    checkResName , //"deployments",
					Subresource: checkSubResName,
					Name:        checkResInstanceName ,
				},
			},
		}

	}else{
		// NonResourceURL
		log("check for NonResource %s \n", checkResName )

		sar = authorizationv1.SubjectAccessReview{  // https://godoc.org/k8s.io/api/authorization/v1#SubjectAccessReview
			Spec: authorizationv1.SubjectAccessReviewSpec{   // https://godoc.org/k8s.io/api/authorization/v1#SubjectAccessReviewSpec
				User: userName ,  // check for a user
				Groups: userGroupName , // check for a user group
				//UID: "" ,
				NonResourceAttributes: &authorizationv1.NonResourceAttributes{  
					// https://godoc.org/k8s.io/api/authorization/v1#NonResourceAttributes
					Path:   checkResName ,
					Verb:   string(checkVerb) ,
				},
			},
		}

	}

	ctx, _ := context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 

	// https://godoc.org/k8s.io/client-go/kubernetes/typed/authorization/v1#AuthorizationV1Client.SubjectAccessReviews
	// https://godoc.org/k8s.io/client-go/kubernetes/typed/authorization/v1#SubjectAccessReviewExpansion
	response, err := client.SubjectAccessReviews().Create(ctx , &sar , metav1.CreateOptions{} )
	if err != nil {
		e=err
		return 
	}


	// response  https://godoc.org/k8s.io/api/authorization/v1#SubjectAccessReview
	log("%v \n" , response )
	
	// https://godoc.org/k8s.io/api/authorization/v1#SubjectAccessReviewStatus
	log("%v \n" , response.Status.Allowed )
	log("%v \n" , response.Status.Reason )
	log("%v \n" , response.Status.EvaluationError )


	allowed=response.Status.Allowed
	reason=response.Status.Reason

	return 
}



func (c *K8sClient)CheckSelfRole(  checkVerb VerbType ,	checkResName string, checkSubResName string ,checkResInstanceName string , checkResApiGroup string , checkResNamespace string ) ( allowed bool , reason string , e error ){

	e=nil
	allowed=false
	reason=""

	if checkVerb==VerbNone {
		e=fmt.Errorf("verb is none " )
		return
	}

	if c.Config == nil {
		if e1:=c.autoConfig() ; e1 !=nil {
			e=fmt.Errorf("failed to config : %v " , e )
			return 
		}
	}

	if len(checkResName)==0 {
		e=fmt.Errorf("empty checkResName " )
		return
	}


	//  https://godoc.org/k8s.io/client-go/kubernetes/typed/authorization/v1#NewForConfig
	client, e1 := authorizationClientv1.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}

	var sar *authorizationv1.SelfSubjectAccessReview

	if strings.HasPrefix(checkResName, "/")==false {
		// ResourceURL
		log("check for Resource %s \n", checkResName )

		sar = &authorizationv1.SelfSubjectAccessReview{  // https://godoc.org/k8s.io/api/authorization/v1#SelfSubjectAccessReview
			Spec: authorizationv1.SelfSubjectAccessReviewSpec{   // https://godoc.org/k8s.io/api/authorization/v1#SelfSubjectAccessReviewSpec
				ResourceAttributes: &authorizationv1.ResourceAttributes{  // https://godoc.org/k8s.io/api/authorization/v1#ResourceAttributes
					Namespace:   checkResNamespace ,
					Verb:        string(checkVerb) ,
					Group:       checkResApiGroup , //"extensions",  // resource api group 
					Version:	"" , //  version of resource api group
									    // Version is the API Version of the Resource.  "*" means all.
									    // +optional
					Resource:    checkResName , //"deployments",
					Subresource: checkSubResName,
					Name:        checkResInstanceName ,
				},
			},
		}

	}else{
		// NonResourceURL
		log("check for NonResource %s \n", checkResName )

		sar = &authorizationv1.SelfSubjectAccessReview{  // https://godoc.org/k8s.io/api/authorization/v1#SelfSubjectAccessReview
			Spec: authorizationv1.SelfSubjectAccessReviewSpec{   // https://godoc.org/k8s.io/api/authorization/v1#SelfSubjectAccessReviewSpec
				NonResourceAttributes: &authorizationv1.NonResourceAttributes{   // https://godoc.org/k8s.io/api/authorization/v1#NonResourceAttributes
					Path:   checkResName ,
					Verb:   string(checkVerb) ,
				},
			},
		}
	}

	ctx, _ := context.WithTimeout(context.Background(), RequestTimeOut*time.Second) 


	// https://godoc.org/k8s.io/client-go/kubernetes/typed/authorization/v1#AuthorizationV1Client.SelfSubjectAccessReviews
	// https://godoc.org/k8s.io/client-go/kubernetes/typed/authorization/v1#SelfSubjectAccessReviewExpansion
	response, err := client.SelfSubjectAccessReviews().Create( ctx , sar , metav1.CreateOptions{} )
	if err != nil {
		e=err
		return 
	}

	// response  https://godoc.org/k8s.io/api/authorization/v1#SubjectAccessReview
	log("%v \n" , response )
	
	// https://godoc.org/k8s.io/api/authorization/v1#SubjectAccessReviewStatus
	log("%v \n" , response.Status.Allowed )
	log("%v \n" , response.Status.Reason )
	log("%v \n" , response.Status.EvaluationError )


	allowed=response.Status.Allowed
	reason=response.Status.Reason

	return 
}


//----------------- informer -----------------------

// informer 就是使用了 K8S的 watch机制，把关系的K8S资源 同步到本地的缓存中，实现查看
//example: 
// https://github.com/kubernetes/client-go/blob/3d5c80942cce510064da1ab62c579e190a0230fd/metadata/metadatainformer/informer_test.go
// https://github.com/kubernetes/client-go/blob/af50d22222d331aaeee988a60a0707a4a4abaf26/examples/fake-client/main_test.go
//  https://github.com/kubernetes/sample-controller/blob/master/main.go
// resourceType: https://github.com/kubernetes/client-go/blob/be97aaa976ad58026e66cd9af5aaf1b006081f09/informers/generic.go#L87
func (c *K8sClient)CreateInformer(  resourceType  schema.GroupVersionResource ,  EventHandlerFuncs *cache.ResourceEventHandlerFuncs )  ( lister cache.GenericLister , stopWatchCh chan struct{} ,  e error ) {

	if c.Config == nil {
		if e1:=c.autoConfig() ; e1 !=nil {
			e=fmt.Errorf("failed to config : %v " , e )
			return 
		}
	}


	Client, e1 := kubernetes.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}

    stopWatchCh = make(chan struct{})

	//=======================================

	// https://github.com/kubernetes/client-go/blob/af50d22222d331aaeee988a60a0707a4a4abaf26/informers/factory.go#L110
	// 第二个参数，是resync ， 如果不为0，就会定期去 list， 即使被监控对象没发生变化，发现都会被定期调用 UpdateFunc 回调 
	kubeInformerFactory:=kubeinformers.NewSharedInformerFactoryWithOptions(Client , time.Second*0 ) 


	//=======================================

	// 动态生成指定 资源类型的 informer
	// https://github.com/kubernetes/client-go/blob/master/informers/factory.go#L187
	// https://github.com/kubernetes/client-go/blob/be97aaa976ad58026e66cd9af5aaf1b006081f09/informers/generic.go#L87
	GenericInformer, er := kubeInformerFactory.ForResource( resourceType )
	// informer 就是 封装了 cache.NewSharedIndexInformer
	if er!=nil {
		e=fmt.Errorf(" failed to get informer for specified resourceType : %v " , er )
		return
	}

	// 静态 生成指定 资源类型的 informer 
	// https://github.com/kubernetes/client-go/blob/af50d22222d331aaeee988a60a0707a4a4abaf26/informers/factory.go#L188
	// podInformer := informers.Core().V1().Pods().Informer()
	// podInformer.AddEventHandler( ...  )


	//=======================================

	// watch发生事件时，可使用回调handler
	if EventHandlerFuncs!=nil {
		// https://github.com/kubernetes/client-go/blob/be97aaa976ad58026e66cd9af5aaf1b006081f09/informers/generic.go#L76
		// https://godoc.org/k8s.io/client-go/tools/cache#SharedIndexInformer
		// https://godoc.org/k8s.io/client-go/tools/cache#ResourceEventHandlerFuncs
		// https://github.com/kubernetes/client-go/blob/master/tools/cache/shared_informer.go#L137
		GenericInformer.Informer().AddEventHandler( EventHandlerFuncs)
	}

	//=======================================

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopWatchCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	// 启动 informer，list & watch
	// stopWatchCh 关闭时，停止运行
	kubeInformerFactory.Start(stopWatchCh)

	//=======================================
    // 从 apiserver 同步资源，即 list 
    // https://github.com/kubernetes/client-go/blob/425ea3e5d030326fecb2994e026a4ead72cadef3/metadata/metadatainformer/informer.go#L97
	//if synced := kubeInformerFactory.WaitForCacheSync(  stopWatchCh  )  ; synced[resourceType] == false {
	if ! cache.WaitForCacheSync( stopWatchCh , GenericInformer.Informer().HasSynced) {
		e=fmt.Errorf(" failed to WaitForCacheSync for specified resourceType "  )
		return
	}

	//=======================================

	// 生成 获取 本地cache 资源数据的 接口
	// https://github.com/kubernetes/client-go/blob/be97aaa976ad58026e66cd9af5aaf1b006081f09/informers/generic.go#L81
	// https://godoc.org/k8s.io/client-go/tools/cache#GenericLister
	lister=GenericInformer.Lister()


    return 

}









