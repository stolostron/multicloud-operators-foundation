package util

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

func CreateClusterRole(kubeClient kubernetes.Interface, name string, rules []rbacv1.PolicyRule) error {
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Rules: rules,
	}

	_, err := kubeClient.RbacV1().ClusterRoles().Create(context.TODO(), clusterRole, metav1.CreateOptions{})
	return err
}

func UpdateClusterRole(kubeClient kubernetes.Interface, name string, rules []rbacv1.PolicyRule) error {
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		clusterRole, err := kubeClient.RbacV1().ClusterRoles().Get(context.TODO(), name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return CreateClusterRole(kubeClient, name, rules)
		case err != nil:
			return err
		}

		clusterRole.Rules = rules
		_, err = kubeClient.RbacV1().ClusterRoles().Update(context.TODO(), clusterRole, metav1.UpdateOptions{})
		return err
	})

	return err
}

func CreateClusterRoleBindingForUser(kubeClient kubernetes.Interface, name, clusterRole, user string) error {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "User",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     user,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRole,
		},
	}

	_, err := kubeClient.RbacV1().ClusterRoleBindings().Create(context.TODO(), clusterRoleBinding, metav1.CreateOptions{})

	return err
}

func DeleteClusterRoleBinding(kubeClient kubernetes.Interface, name string) error {
	err := kubeClient.RbacV1().ClusterRoleBindings().Delete(context.TODO(), name, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

func DeleteClusterRole(kubeClient kubernetes.Interface, name string) error {
	err := kubeClient.RbacV1().ClusterRoles().Delete(context.TODO(), name, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

func CreateClusterRoleBinding(kubeClient kubernetes.Interface, name, clusterRole, user string) error {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRole,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "User",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     user,
			},
		},
	}
	_, err := kubeClient.RbacV1().ClusterRoleBindings().Create(context.TODO(), clusterRoleBinding, metav1.CreateOptions{})
	return err
}

func CreateFakeTlsSecret(kubeClient kubernetes.Interface, name, namespace string) (*corev1.Secret, error) {
	tlsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"tls.crt": []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURWekNDQWorZ0F3SUJBZ0lKQU5mTC9CUjgrejZGTUEwR0NTcUdTSWIzRFFFQkN3VUFNRUl4Q3pBSkJnTlYKQkFZVEFsaFlNUlV3RXdZRFZRUUhEQXhFWldaaGRXeDBJRU5wZEhreEhEQWFCZ05WQkFvTUUwUmxabUYxYkhRZwpRMjl0Y0dGdWVTQk1kR1F3SGhjTk1qRXdOek13TURNeU5ERXhXaGNOTWpJd056TXdNRE15TkRFeFdqQkNNUXN3CkNRWURWUVFHRXdKWVdERVZNQk1HQTFVRUJ3d01SR1ZtWVhWc2RDQkRhWFI1TVJ3d0dnWURWUVFLREJORVpXWmgKZFd4MElFTnZiWEJoYm5rZ1RIUmtNSUlCSWpBTkJna3Foa2lHOXcwQkFRRUZBQU9DQVE4QU1JSUJDZ0tDQVFFQQpwWU5vSXRxODh1eG9CVWp1U2swa2svTjlIdC9BQlI3WlB1bVEyS0NvWitGQTF2RGE3RmUvRk8waGVBdzhyT2RYCnJaQUUzM2FoOENCVXFOTFk3K2lrRzdmUTNpSlpxSEF2eGEvVnFKQi9YcDJuazlRTGd5U1dZZllDNDE1ZlhYZjMKYUdFbXJZNnV6bmhJZFZmdWp4T3NuK0ZHcS9qMGVHemhWSGNJczRtcStPWk1UMENXRjNkYzN0VjF1VGF2Rk1hdgp3UFlxODJJdFd6UThrdTBSTEpaMGoraWVSNTFDY1Y4aFVUOWlyVXFBb1pEUlBFK0hsZmo1aUU1cUdTNERHK2IrCkNoaGZ0a3AyNjF2YUN3b0cyYnBsMnpoLzlEQXJucm9hbHkzM3BFN2F2blk0Vyt1YkNmMURPWmlqWGFwQTlodFIKOVduNW9IS1AydTBmbXd6ejBNVzUyUUlEQVFBQm8xQXdUakFkQmdOVkhRNEVGZ1FVakp0KzdRcG82emZNcGljOApVKzRLUTRmZURBVXdId1lEVlIwakJCZ3dGb0FVakp0KzdRcG82emZNcGljOFUrNEtRNGZlREFVd0RBWURWUjBUCkJBVXdBd0VCL3pBTkJna3Foa2lHOXcwQkFRc0ZBQU9DQVFFQUxvMHg4K2dBNlp6SlJJM05LZDNBbW5NdXhGNVcKY29qUDk1WGFyNEN0WUN0dDV4MS9tZnNGSmRVWnNwTmprSGtrQVBMK3JBb3hRVFBYSklPem5qaTZzQVdqSGRKUgplYXpVOGtKcCtHcVdpbHo1eXNMVk8vRS9BeFp1MlpPUStsSlJCZC93UjlXTUhVRVpOa3pCK09aR2JSTDNXUjhCCjcvVk8wVFpYcTZQM0M1dG1WSzdKZUFoNzVyejlKV3ZZNmNyNExMa05jaWtEUy96dmpjT3BFc0lGQkp3UGpMbGcKVG1qcElWcGs0czhSek5UMVJRVzgxQUw1aEYyTit0Zzc5eERIdi9KZ3I4dnJYUi96clowZ0dCRXQ4SXV3STdBRApDeG5sWTFsSFRSTHltY1NmYlNqVTQ1cEpDNWVkYVNWelR0N0thVzM2VkozK1BnVGkvaE1iNDZwN1BRPT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="),
			"tls.key": []byte("LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb2dJQkFBS0NBUUVBcFlOb0l0cTg4dXhvQlVqdVNrMGtrL045SHQvQUJSN1pQdW1RMktDb1orRkExdkRhCjdGZS9GTzBoZUF3OHJPZFhyWkFFMzNhaDhDQlVxTkxZNytpa0c3ZlEzaUpacUhBdnhhL1ZxSkIvWHAybms5UUwKZ3lTV1lmWUM0MTVmWFhmM2FHRW1yWTZ1em5oSWRWZnVqeE9zbitGR3EvajBlR3poVkhjSXM0bXErT1pNVDBDVwpGM2RjM3RWMXVUYXZGTWF2d1BZcTgySXRXelE4a3UwUkxKWjBqK2llUjUxQ2NWOGhVVDlpclVxQW9aRFJQRStICmxmajVpRTVxR1M0REcrYitDaGhmdGtwMjYxdmFDd29HMmJwbDJ6aC85REFybnJvYWx5MzNwRTdhdm5ZNFcrdWIKQ2YxRE9aaWpYYXBBOWh0UjlXbjVvSEtQMnUwZm13enowTVc1MlFJREFRQUJBb0lCQUN5VDF0RkVYbzJDMUlWUAozallPenVJMk95VzhsNmdKWmZPR3pxYzVwZ0hNYmowMXc1RFNGVG5hb0NBSUU3TngzM0Iwa0l0ckZUUnFVTUxqCmZ1QW1wVVI5M25obGdnWldxTmN5ZzNZUjdPd2J4QTJSbDhRcmI0RlUwL1JPNzVwcC9DMlZ0T2didk1NSkxHTEcKV3c4WCttOVpLa0taRzZidmxFUytobzVzYnFyNFZRVnFsV1dUbXBKckRScUNPanZGYlgwQkJQd0x1d2E2bFNFRgpNQ0RrZWhEZkQzbWh5TFNXaVFYb0xoOXljSEpXY0hhSVBqWDVPaFVyQ1BLZVJKcDZ3SDA0a3Ztc1p0R2pMRGlPCjBxQmRxSm5SSVRXYXRpNlJsVldPcUE2R2ZUcHhXVmljcGlpbHYvcUJBRG12NlB0UTZzaHZTQTBQSWdYRFRhM08KcDdTNGZURUNnWUVBenNWdHJvcjdLc3FoaE85VlpqcVpyQ0Y5cC9TZmU2V3Nia2NwY2lvRDlnZ25uSmJVdDhQWgozNzNQQ2RFMzFMQWtmRjZiQTZSRjJSL1UvU20wOUM0bkozRksvdXI2R2hMelh6OE5OMElNanAzV2NTM3JmN01CCkhRd3BOdXNSRC9HRjBzVm82bjh5eU5KL2tmSXE3ZThGd0txcGplWlZvZTBvd3NFRGNoTGtTclVDZ1lFQXpPdFcKREJTZytXSjd2VTdNazlJTFdSSWtXVFczMkRXRHNGM2NIWUg5bHBKcGdKMVE1ZVlqYjZsU1V4ZTdYTkRLeDkwMwpYbnFJZ0ExdlFjYzh2RCtTa3RLZ1lyZzFHb1ZHVEJJWXNzVjl3eXkxUDczdUQya0ZhUTRPb3FCWWQ3Sm5ab2YrCkhmc1FNNWZqUTZaWnJhQ1hDY2dJQkdaUFBaajlZajdSRFZFMzFSVUNnWUIyNUJocit5ZnVjL0twa0VBbmR0eHoKbUJJN1o3SG9FOXZ3ME9RbzY3VzVXdmtEMWNwY0c3WUVLNHlIVlpCbnNCeGFrcjlKT2NTYjB1elI0SkJXc3M3NgpvKzcrWXJnS0ZBbHlJN3dDb095OWVFNGNaODM0Y0VIY3BPaHgxbm5LRkJMaG5YYjFGc3hwb25lTndKUWttWUpTClJROFhNM0RibVpVTlhwUVBuSU05M1FLQmdDWXBpYXZVUjZwSjhmdHVhbUQ1RkEzeGQvMTVLSlRHV3BFRTJkSlEKL0JZSGpFaGNnODFjejZxaTRPY0NtMjBNb1VjWlpvOWN5SUQ0ZjRqRGZ3Y2IyOE1tSUtKaDVkbmJpaHp1bmRUbApQS2VWY3VlOUNsR3FZRXlSUnA1NHVDRUtnNEV2d0Y1Ni9DaHZsKzVvVTNrbldCbUZQQ0Q5b0xJN0JLMUFQNVI0ClZLcUZBb0dBY3lwZ3d4c0o2bFhpU1lxdCtTSiswU2I1cG1VTHJqajFGcGpaeGkydkMxUkhkY0xlWTliZmdPS2UKZHZiL2hUWThpK3NaRFFiRzZqY01ZQmlVSDErUk50MnlJM25Bb0hkcmxkYWd1YVJiSlNIV254OU02eWxsZnpSaQpDQVNBM0hkakVKaWtFTUw1MEFncUw0WjNxQjc2ZWwxZXRmREJ1QWJGcFNsaDI2MU9meDQ9Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg=="),
		},
		Type: corev1.SecretTypeTLS,
	}
	return kubeClient.CoreV1().Secrets(namespace).Create(context.TODO(), tlsSecret, metav1.CreateOptions{})
}

func CreateFakeRootCaConfigMap(kubeClient kubernetes.Interface, name, namespace string) (*corev1.ConfigMap, error) {
	rootCa := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"ca.crt": "fake-ca-data",
		},
	}
	return kubeClient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), rootCa, metav1.CreateOptions{})
}
