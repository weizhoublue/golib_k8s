package golib_k8s
import (
	"os"
	"fmt"
	"os/signal"
	"path/filepath"
	"regexp"
	goRuntime "runtime"
	"strings"
	"strconv"
	"context"
	"syscall"
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

	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/leaderelection"
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



//--------------------- namesapces

func (c *K8sClient)GetNamespace(  ) (  nsList []string , nsDetailList []corev1.Namespace , e error )

//--------------------- node

func (c *K8sClient)GetNodes(  ) (  nodeList []map[string]string , nodeDetailList []corev1.Node , e error )


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



//-------------- lease
// 使用 k8s 的 lease 资源，实现 leader 选举
// 注意：无论几个候选人上来，用什么样的ip，只要是 myId 相同， 他们都会拿到leader ， 简单说，只认 myId
//如果持续 leaseDuration 没有 续租，则会丢失leader
// renewDeadline 获取leader 后， 自动续租的周期
// retryLockPeriod 尝试获取 leader 的间隔
// 当取消leader选举，后者退出leader角色，务必调用 cancelLease  ， 否则 RunOrDie 协程 泄漏
func (c *K8sClient)Lease( leaseLockName , leaseLockNamespace , myId string ,
	leaseDuration , renewDeadline , retryLockPeriod uint , newLeaderHandler func(identity string)  )  ( acquireLeaderChan , lostLeaderChan chan bool , cancelLease context.CancelFunc ,  e error ) {


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
		t:=time.Now().Format(time.RFC3339Nano)
        return fmt.Printf( fmt.Sprintf("[%v] %v %v" , t, prefix , format ) , a... )
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

	// too long
	//log("%v \n" , config )

	c.Config=config

	return nil
}


//=============namespace=======================

/*
output:
	nsList [ namespace_name ]
	nsDetailList []Namespace  // struct defination: https://godoc.org/k8s.io/api/core/v1#Namespace
*/
func (c *K8sClient)GetNamespace(  ) (  nsList []string , nsDetailList []corev1.Namespace , e error ){

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

	// https://godoc.org/k8s.io/client-go/kubernetes/typed/core/v1#CoreV1Client.Namespaces
	// https://github.com/kubernetes/client-go/blob/master/kubernetes/typed/core/v1/namespace.go#L40
	info , err := client.CoreV1().Namespaces().List( ctx ,  metav1.ListOptions{})
	if err != nil {
		e=fmt.Errorf("%v" , err )
		return
	}
	if info==nil {
		return
	}

	for _ , v := range info.Items {

		nsList=append(nsList , v.ObjectMeta.Name )
	}

 	nsDetailList=info.Items

	return
}



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
			return fmt.Errorf( "failed to get deploymen=%v , info=%v " , deploymentName , e )
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
			return fmt.Errorf( "failed to get deploymen=%v , info=%v " , deploymentName , e )
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



//------ lease ----
/*
//原理例子  https://silenceper.com/blog/202002/kubernetes-leaderelection/
基本原理其实就是利用通过Kubernetes中 configmap ， endpoints 或者 lease 资源实现一个分布式锁，抢(acqure)到锁的节点成为leader，
并且定期更新（renew）。其他进程也在不断的尝试进行抢占，抢占不到则继续等待下次循环。当leader节点挂掉之后，租约到期，其他节点就成为新的leader
在 k8s 的 client-go 的 api 中，上面的锁叫做 resourcelock，现在 client-go 中支持四种 resourceLock:
configMapLock：基于 configMap 资源的操作扩展的分布式锁
endpointLock: 基于 endPoint 资源的操作扩展的分布式锁
leaseLock：基于 lease 资源，相对来说 leader 资源比较轻量
multiLock：多种锁混合使用，一个 multilock ，即可以进行选择两种，当其中一种保存失败时，选择第二张
*/


// https://pkg.go.dev/k8s.io/api@v0.22.4/coordination/v1
// https://pkg.go.dev/k8s.io/client-go/tools/leaderelection
// https://pkg.go.dev/k8s.io/client-go@v0.22.4/tools/leaderelection/resourcelock

// 使用 k8s 的 lease 资源，实现 leader 选举
// 注意：无论几个候选人上来，用什么样的ip，只要是 myId 相同， 他们都会拿到leader ， 简单说，只认 myId
//如果持续 leaseDuration 没有 续租，则会丢失leader
// renewDeadline 获取leader 后， 自动续租的周期
// retryLockPeriod 尝试获取 leader 的间隔
// 当取消leader选举，后者退出leader角色，务必调用 cancelLease  ， 否则 RunOrDie 协程 泄漏
func (c *K8sClient)Lease( leaseLockName , leaseLockNamespace , myId string ,
	leaseDuration , renewDeadline , retryLockPeriod uint , newLeaderHandler func(identity string)  )  ( acquireLeaderChan , lostLeaderChan chan bool , cancelLease context.CancelFunc ,  e error ) {

	// lease 名字必须是小写的，由数字，字母，'-' or '.' 等组成
	// a lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters, '-' or '.',
	// and must start and end with an alphanumeric character
	// (e.g. 'example.com', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')
	if len(leaseLockName)==0 {
		e=fmt.Errorf("leaseLockName is empty "  )
		return
	}
	re := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
	if re.MatchString(leaseLockName)==false {
		e=fmt.Errorf("leaseLockName is not ignore "  )
		return
	}

	if len(leaseLockNamespace)==0 {
		e=fmt.Errorf("leaseLockNamespace is empty "  )
		return
	}
	if len(myId)==0 {
		e=fmt.Errorf("myId is empty "  )
		return
	}
	if leaseDuration==0 {
		leaseDuration=15
		log("adjust leaseDuration to %v \n", leaseDuration )
	}
	if renewDeadline==0 {
		renewDeadline=10
		log("adjust renewDeadline to %v \n", renewDeadline )
	}
	if retryLockPeriod==0 {
		retryLockPeriod=2
		log("adjust retryLockPeriod to %v \n", retryLockPeriod )
	}

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


	// use a Go context so we can tell the leaderelection code when we
	// want to step down
	var ctx context.Context
	ctx, cancelLease = context.WithCancel(context.Background())


	acquireLeaderChan=make(chan bool)
	lostLeaderChan=make(chan bool, 10 )

	// we use the Lease lock type since edits to Leases are less common
	// and fewer objects in the cluster watch "all Leases".
	// 指定锁的资源对象，这里使用了Lease资源，还支持configmap，endpoint，或者multilock(即多种配合使用)
	// https://pkg.go.dev/k8s.io/client-go@v0.22.4/tools/leaderelection/resourcelock#LeaseLock
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      leaseLockName,
			Namespace: leaseLockNamespace,
		},
		Client: client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: myId,
		},
	}


	// listen for interrupts or the Linux SIGTERM signal and cancel
	// our context, which the leader election code will observe and
	// step down
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-ch:
			log("Received termination, stop leader %s for leaase %v/%v \n", myId, leaseLockNamespace, leaseLockName)
			cancelLease()
		case <-ctx.Done():
			log("cancel leader %s for leaase %v/%v \n", myId, leaseLockNamespace, leaseLockName)
		}
	}()


	go func() {
		// https://pkg.go.dev/k8s.io/client-go/tools/leaderelection@v0.22.4#RunOrDie
		// https://pkg.go.dev/k8s.io/client-go/tools/leaderelection@v0.22.4#LeaderElectionConfig

		// RunOrDie blocks until leader election loop is stopped by ctx or it has stopped holding the leader lease
		leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
			Lock: lock,
			// IMPORTANT: you MUST ensure that any code you have that
			// is protected by the lease must terminate **before**
			// you call cancel. Otherwise, you could have a background
			// loop still running and another process could
			// get elected before your background loop finished, violating
			// the stated goal of the lease.
			ReleaseOnCancel: true,

			// LeaseDuration is the duration that non-leader candidates will
			// wait to force acquire leadership. This is measured against time of
			// last observed ack.
			//
			// A client needs to wait a full LeaseDuration without observing a change to
			// the record before it can attempt to take over. When all clients are
			// shutdown and a new set of clients are started with different names against
			// the same leader record, they must wait the full LeaseDuration before
			// attempting to acquire the lease. Thus LeaseDuration should be as short as
			// possible (within your tolerance for clock skew rate) to avoid a possible
			// long waits in the scenario.
			//
			// Core clients default this value to 15 seconds.
			//租约时间
			LeaseDuration: time.Duration(leaseDuration)*time.Second , //租约时间

			// RenewDeadline is the duration that the acting master will retry
			// refreshing leadership before giving up.
			// Core clients default this value to 10 seconds.
			// leader 持有 leadse 的 续约周期
			RenewDeadline: time.Duration(renewDeadline) * time.Second, //更新租约的

			// RetryPeriod is the duration the LeaderElector clients should wait
			// between tries of actions.
			// Core clients default this value to 2 seconds.
			// 所有候选人 尝试进行 竞选 尝试的间隔
			RetryPeriod: time.Duration(retryLockPeriod) * time.Second,

			//https://pkg.go.dev/k8s.io/client-go/tools/leaderelection@v0.22.4#LeaderCallbacks
			Callbacks: leaderelection.LeaderCallbacks{

				// when we get the leader, we could do business
				// OnStartedLeading is called when a LeaderElector client starts leading
				OnStartedLeading: func(ctx context.Context) {
					//变为leader执行的业务代码
					// we're notified when we start - this is where you would
					// usually put your code
					log("acquire leader %s for leaase %v/%v \n", myId, leaseLockNamespace, leaseLockName)
					acquireLeaderChan <- true
					//--
					<-ctx.Done()
					log("stop leader %s for leaase %v/%v \n", myId, leaseLockNamespace, leaseLockName)

				},

				// OnStoppedLeading is called when a LeaderElector client stops leading
				// when we release the leader
				OnStoppedLeading: func() {
					// 我们主动放弃 或者异常失去了 leader
					// we can do cleanup here
					log("quit or lost leader %s for leaase %v/%v \n", myId, leaseLockNamespace, leaseLockName)
					select{
						case lostLeaderChan <-true:
						default:
							log("error, failed to inform lostLeaderChan for  leaase %v/%v \n" , leaseLockNamespace, leaseLockName )
					}
				},

				// OnNewLeader is called when the client observes a leader that is
				// not the previously observed leader. This includes the first observed
				// leader when the client starts.
				OnNewLeader: func(identity string) {
					//当产生新的leader后执行的方法
					// we're notified when new leader elected
					if identity != myId {
						log(" leader changed to %s for leaase %v/%v , not my id %v \n", identity, leaseLockNamespace, leaseLockName ,myId )
					}
					if newLeaderHandler != nil {
						newLeaderHandler(identity)
					}
				},
			},
		})
	}()

	return

}



