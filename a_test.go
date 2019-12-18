package golib_k8s_test
import (
	k8s "github.com/weizhouBlue/golib_k8s"
	"testing"
	"fmt"
	"time"


	// for creating deployment 
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"


	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

)




func Test_1(t *testing.T){

	k8s.EnableLog=true
	k:=k8s.K8sClient{}

	err:=k.AutoConfig()
	if err!=nil {
		fmt.Println(  "failed to create k8s client" )
		t.FailNow()
	}

	namespace:="default"
	if podInfo , _ , err:=k.GetPods(namespace) ; err!=nil {
		fmt.Println(  err )
		t.FailNow()
	}else{
		fmt.Printf("podInfo=%v \n" , podInfo)
	}

	podName:="test3-nginx-585dbc7464-txjl6"
	if b , err :=k.CheckPodHealthy( namespace  ,  podName ) ; err !=nil {
		fmt.Println(  err )
		t.FailNow()
	}else{
		fmt.Printf("healthy=%v \n" , b )
	}

}



// recommended method
func Test_2(t *testing.T){

	k8s.EnableLog=true
	k:=k8s.K8sClient{}

	err:=k.AutoConfig()
	if err!=nil {
		fmt.Println(  "failed to create k8s client" )
		t.FailNow()
	}

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
	if deploymentBasicInfo, deploymentDetailInfo , e:=k.GetDeployment( namespace , "" ) ; e!=nil {
		fmt.Println(  err )
		t.FailNow()
	}else{
		fmt.Printf("deploymentBasicInfo=%v\n" , deploymentBasicInfo )

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

	err:=k.AutoConfig()
	if err!=nil {
		fmt.Println(  "failed to create k8s client" )
		t.FailNow()
	}


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
	deploymentName=""
	if deploymentBasicInfo, _ , err:=k.GetDeploymentTyped( namespace , deploymentName ) ; err!=nil {
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

