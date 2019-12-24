package golib_k8s
import (
	"os"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"strconv"

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
)


var (
	//for log
    EnableLog=false
    // for config
    ScInPodPath="/var/run/secrets/kubernetes.io/serviceaccount"
    KubeConfigPath=filepath.Join(os.Getenv("HOME"), ".kube", "config")
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
	    funcName,filepath ,line,ok := runtime.Caller(1)
	    if ok {
	    	file:=getFileName(filepath)
	    	funcname:=getFileName(runtime.FuncForPC(funcName).Name())
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

func (c *K8sClient) AutoConfig( )  error  {

	var config *rest.Config
	var err error 

	if ExistDir(ScInPodPath)==true {
		log("In pod , try to get config from serviceaccount \n")

		config, err = rest.InClusterConfig()
		if err != nil {
			return fmt.Errorf("failed to get config from serviceaccount=%v , info=%v" , ScInPodPath , err )
		}

	}else if existFile(KubeConfigPath)==true {
		log("Outside of pod , try to get config from kube config file \n")

		config, err = clientcmd.BuildConfigFromFlags("", KubeConfigPath)
		if err != nil {
			return fmt.Errorf("failed to get config from kube config=%v , info=%v" , KubeConfigPath , err )
		}

	}else{
		return fmt.Errorf("failed to get config " )
	}

	log("%v \n" , config )

	c.Config=config

	return nil
}


//=============namespace=======================


//=============pod=======================
/*
input:
	namespace="" , get all namespaces
output:
	podNameList [{"Name":.. "Namespace":.. "UID":...}]
	podDetail []Pod  // struct defination: https://godoc.org/k8s.io/api/core/v1#Pod
*/
func (c *K8sClient)GetPods( namespace string ) (  podNameList []map[string]string , podDetailList []corev1.Pod , e error ){

	if c.Config == nil {
		e=fmt.Errorf("struct K8sClient is not initialized correctly , Config==nil " )
		return
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

	//https://godoc.org/k8s.io/client-go/kubernetes/typed/core/v1#PodInterface
	pods, err := client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
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

	for _, k :=range pods.Items {
		podNameList=append(podNameList , map[string]string {
			"Name": k.ObjectMeta.Name ,
			"Namespace": k.ObjectMeta.Namespace ,
			"UID": string(k.ObjectMeta.UID) ,
		} )
	}

	podDetailList=pods.Items
	return 
}


/*
input:
	namespace
*/
func (c *K8sClient)CheckPodHealthy( namespace string , podName string ) (  exist bool , e error ){

	exist=false

	if c.Config == nil {
		e=fmt.Errorf("struct K8sClient is not initialized correctly , Config==nil " )
		return
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

	_, err := Client.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
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




//=============deployment=======================


/*
input:
	namespace    // namespace=""  for all namespace
output:
	deploymentDetailInfo  []appsv1.Deployment  // // type Deployment : https://godoc.org/k8s.io/api/apps/v1#Deployment
*/
func (c *K8sClient)GetDeploymentTyped( namespace string ,  deploymentName string ) (  deploymentBasicInfo []map[string]string , deploymentDetailList  []appsv1.Deployment , e error ){

	if c.Config == nil {
		e=fmt.Errorf("struct K8sClient is not initialized correctly , Config==nil " )
		return
	}

	if len(deploymentName)!=0 && len(namespace)==0 {
		e=fmt.Errorf("namespace could not be empty when deploymentName is set " )
		return	
	}


	Client, e1 := kubernetes.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}

	if len(deploymentName)==0 {
		if len(namespace)==0 {
			log("get all deployment from all namesapces \n")
		}else{
			log("get all deployment from namesapces=%s \n" , namespace )
		}

		// https://godoc.org/k8s.io/client-go/kubernetes/typed/apps/v1#DeploymentInterface 
		// https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#ListOptions
		result, err  := Client.AppsV1().Deployments(namespace).List( metav1.ListOptions{} )
		if err != nil {
			e=fmt.Errorf( "%v", err )
			return
		}

		//type DeploymentList:  https://godoc.org/k8s.io/api/apps/v1#DeploymentList
		log("TypeMeta=%v \n" , result.TypeMeta )
		log("ListMeta=%v \n" , result.ListMeta )
		log("Deployment number=%v \n", len(result.Items)  )

		for _, v :=range result.Items {
			// type Deployment : https://godoc.org/k8s.io/api/apps/v1#Deployment
			deploymentBasicInfo=append(deploymentBasicInfo , map[string]string{
				"Name": v.ObjectMeta.Name ,
				"Namespace": v.ObjectMeta.Namespace ,
				"UID": string(v.ObjectMeta.UID) ,
			})
		}
		deploymentDetailList=result.Items

	}else{
		if len(namespace)==0 {
			log("get deployment %v from all namesapces \n" , deploymentName )
		}else{
			log("get deployment %v from namesapces=%s \n" , deploymentName , namespace )
		}

		// https://godoc.org/k8s.io/client-go/kubernetes/typed/apps/v1#DeploymentInterface 
		// https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#ListOptions
		v, err  := Client.AppsV1().Deployments(namespace).Get( deploymentName ,  metav1.GetOptions{} )
		if err != nil {
			e=fmt.Errorf( "%v", err )
			return
		}

		deploymentBasicInfo=append(deploymentBasicInfo , map[string]string{
			"Name": v.ObjectMeta.Name ,
			"Namespace": v.ObjectMeta.Namespace  ,
			"UID" : string(v.ObjectMeta.UID) ,
		})
		deploymentDetailList=append(deploymentDetailList, *v )

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

func (c *K8sClient)GetDeployment( namespace string ,  deploymentName string ) (  deploymentBasicInfo []map[string]string , deploymentDetailList  []unstructured.Unstructured , e error ){

	if c.Config == nil {
		e=fmt.Errorf("struct K8sClient is not initialized correctly , Config==nil " )
		return
	}

	if len(deploymentName)!=0 && len(namespace)==0 {
		e=fmt.Errorf("namespace could not be empty when deploymentName is set " )
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
	if len(deploymentName)==0 {

		if len(namespace)==0 {
			log("get all deployment from all namesapces \n")
		}else{
			log("get all deployment from namesapces=%s \n" , namespace )
		}

		// https://godoc.org/k8s.io/api/apps/v1#Resource
		// https://godoc.org/k8s.io/client-go/dynamic#Interface
		// https://godoc.org/k8s.io/client-go/dynamic#NamespaceableResourceInterface
		list, err := Client.Resource(deploymentRes).Namespace(namespace).List(metav1.ListOptions{})
		if err != nil {
			e=fmt.Errorf(" info=%v  " , err  )
			return
		}

		//for list struct : https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured#UnstructuredList
		// log("%v \n" , list )
		log("%v \n" , list.Object )

		for _, d := range list.Items {
			//for list.Items struct:  https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured#Unstructured

				//get deployment name
				name:=d.GetName()
				//get GetNamespace name
				namespace:=d.GetNamespace()
				//get GetNamespace name
				uid:=d.GetUID()

				//a general method to get whatever information you want from the deplyment yaml
				// unstructured:  https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured
				// replicas, found, err := unstructured.NestedInt64(d.Object, "spec", "replicas")
				// if err != nil || !found {
				// 	fmt.Printf("Replicas not found for deployment %s: error=%s", name , err)
				// }else{
				// 	log("%v have replicas=%v\n", name , replicas )
				// }

				deploymentBasicInfo=append(deploymentBasicInfo , map[string]string{
					"Name": name ,
					"Namespace": namespace ,
					"UID" : string(uid) ,
				})

		}

		deploymentDetailList=list.Items

	}else{

		if len(namespace)==0 {
			log("get deployment %v from all namesapces \n" , deploymentName )
		}else{
			log("get deployment %v from namesapces=%s \n" , deploymentName , namespace )
		}

		// https://godoc.org/k8s.io/api/apps/v1#Resource
		// https://godoc.org/k8s.io/client-go/dynamic#NamespaceableResourceInterface
		d, err := Client.Resource(deploymentRes).Namespace(namespace).Get(deploymentName, metav1.GetOptions{})
		if err != nil {
			e=fmt.Errorf(" info=%v  " , err  )
			return
		}

		deploymentBasicInfo=append(deploymentBasicInfo , map[string]string{
			"Name": d.GetName() ,
			"Namespace": d.GetNamespace() ,
			"UID" : string(d.GetUID()) ,
		})

		deploymentDetailList=append(deploymentDetailList, *d)
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
		e=fmt.Errorf("struct K8sClient is not initialized correctly , Config==nil " )
		return
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


	//type DeploymentInterface:  https://godoc.org/k8s.io/client-go/kubernetes/typed/apps/v1#DeploymentInterface
	info , err := Client.AppsV1().Deployments(namespace).Create(deploymentSpec) 
	if err != nil {
		e=fmt.Errorf( "%v", err )
		return
	}

	log("succeeded  to create deployment: %v \n" , info )

	return 
}




func (c *K8sClient)CreateDeployment( namespace string , deploymentYaml *unstructured.Unstructured ) ( e error ){

	if c.Config == nil {
		e=fmt.Errorf("struct K8sClient is not initialized correctly , Config==nil " )
		return
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

	// https://godoc.org/k8s.io/apimachinery/pkg/runtime/schema
	// https://godoc.org/k8s.io/apimachinery/pkg/runtime/schema#GroupVersionResource
	deploymentRes := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	// https://godoc.org/k8s.io/api/apps/v1#Resource
	// https://godoc.org/k8s.io/client-go/dynamic#NamespaceableResourceInterface
	result, err := Client.Resource(deploymentRes).Namespace(namespace).Create(deploymentYaml, metav1.CreateOptions{})
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
		e=fmt.Errorf("struct K8sClient is not initialized correctly , Config==nil " )
		return
	}


	Client, e1 := kubernetes.NewForConfig(c.Config)
	if e1 != nil {
		e=fmt.Errorf("failed to NewForConfig, info=%v , config=%v " , e1 , c.Config )
		return
	}


	// https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#DeletionPropagation
	deletePolicy := metav1.DeletePropagationForeground

	//type DeploymentInterface:  https://godoc.org/k8s.io/client-go/kubernetes/typed/apps/v1#DeploymentInterface
	err := Client.AppsV1().Deployments(namespace).Delete(deploymentName , 
			// https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#DeleteOptions
			&metav1.DeleteOptions{
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
		e=fmt.Errorf("struct K8sClient is not initialized correctly , Config==nil " )
		return
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
	deleteOptions := &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}
	// https://godoc.org/k8s.io/api/apps/v1#Resource
	// https://godoc.org/k8s.io/client-go/dynamic#NamespaceableResourceInterface	
	if err := Client.Resource(deploymentRes).Namespace(namespace).Delete(deploymentName, deleteOptions); err != nil {
		e=fmt.Errorf(" info=%v  " , err  )
		return
	}

	log("succeeded to delete %v \n" , deploymentName )
	return 
}




func (c *K8sClient)UpdateDeploymentTyped( namespace string , deploymentName string , handler func(*appsv1.Deployment)error  ) ( e error ){

	if c.Config == nil {
		e=fmt.Errorf("struct K8sClient is not initialized correctly , Config==nil " )
		return
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
		_, deploymentDetailList , e:=c.GetDeploymentTyped( namespace , deploymentName ) 
		if e!=nil {
			return fmt.Errorf( "failed to get deploymen=%v , info=%v " , e )
		}
		result:=deploymentDetailList[0]
		log("got deployment %v yaml under namespace %v  \n" , deploymentName  , namespace )

		//log("%v\n" , result )
		if err:=handler(&result) ; err!=nil {
			return fmt.Errorf( "failed to call hander info=%v " , err )
		}
		//log("%v\n" , result )


		//type DeploymentInterface:  https://godoc.org/k8s.io/client-go/kubernetes/typed/apps/v1#DeploymentInterface
		_, err := Client.AppsV1().Deployments(namespace).Update(&result )
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
		e=fmt.Errorf("struct K8sClient is not initialized correctly , Config==nil " )
		return
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
		_, deploymentDetailList , e:=c.GetDeployment( namespace , deploymentName ) 
		if e!=nil {
			return fmt.Errorf( "failed to get deploymen=%v , info=%v " , e )
		}
		result:=deploymentDetailList[0]
		log("got deployment %v yaml under namespace %v  \n" , deploymentName  , namespace )

		//log("%v\n" , result )
		if err:=handler(&result) ; err!=nil {
			return fmt.Errorf( "failed to call hander info=%v " , err )
		}
		//log("%v\n" , result )


		// https://godoc.org/k8s.io/apimachinery/pkg/runtime/schema
		// https://godoc.org/k8s.io/apimachinery/pkg/runtime/schema#GroupVersionResource
		deploymentRes := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
		// https://godoc.org/k8s.io/api/apps/v1#Resource
		// https://godoc.org/k8s.io/client-go/dynamic#NamespaceableResourceInterface
		_, err := Client.Resource(deploymentRes).Namespace(namespace).Update(&result, metav1.UpdateOptions{})
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
	VerbGet=VerbType("get")
	VerbList=VerbType("list")
	VerbWatch=VerbType("watch")
	VerbCreate=VerbType("create")
	VerbUpdate=VerbType("update")
	VerbPatch=VerbType("patch")
	VerbDelete=VerbType("delete")
	VerbDeletecollection=VerbType("deletecollection")
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

	if c.Config == nil {
		e=fmt.Errorf("struct K8sClient is not initialized correctly , Config==nil " )
		return
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

	var sar *authorizationv1.SubjectAccessReview

	if strings.HasPrefix(checkResName, "/")==false {
		// ResourceURL
		log("check for Resource %s \n", checkResName )

		sar = &authorizationv1.SubjectAccessReview{  // https://godoc.org/k8s.io/api/authorization/v1#SubjectAccessReview
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

		sar = &authorizationv1.SubjectAccessReview{  // https://godoc.org/k8s.io/api/authorization/v1#SubjectAccessReview
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

	// https://godoc.org/k8s.io/client-go/kubernetes/typed/authorization/v1#AuthorizationV1Client.SubjectAccessReviews
	// https://godoc.org/k8s.io/client-go/kubernetes/typed/authorization/v1#SubjectAccessReviewExpansion
	response, err := client.SubjectAccessReviews().Create(sar)
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

	if c.Config == nil {
		e=fmt.Errorf("struct K8sClient is not initialized correctly , Config==nil " )
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

	// https://godoc.org/k8s.io/client-go/kubernetes/typed/authorization/v1#AuthorizationV1Client.SelfSubjectAccessReviews
	// https://godoc.org/k8s.io/client-go/kubernetes/typed/authorization/v1#SelfSubjectAccessReviewExpansion
	response, err := client.SelfSubjectAccessReviews().Create(sar)
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






