# Frequently Asked Questions

## What problem does the inventory controller solve?

We assume that each customer has their own set of tools (CSV files, spreadsheets, CMDB, etc.) to manage inventory data and there is not a common data-format that they use to store the data. This operator provides an API for customers to take inventory data from their existing systems in various formats to the multi-cluster inventory system so they are available for use in the baremetal clusters.

It provides a CRD called BaremetalAsset that is used to hold the inventory assets in the hub cluster and a controller that reconciles the assets with resources in the managed cluster. That means each customer will need a custom tool outside of ACM to take inventory data from their existing systems and convert it into a format that BMA CRD wants.

The customer will likely do a bulk-import of inventory records into the multicluster-inventory as BareMetalAssets (BMA). These BMAs do not have to be part of a cluster just yet, and they can just live in the namespace. Once it's time to add the inventory to a cluster, they can assign a Role and ClusterDeployment to the asset and the controller is responsible for updating the managed cluster with the corresponding BareMetalHost resources.

## How does it fit into ACM?

BareMetalAssets (BMAs) are added to multicluster-inventory system either through ACM UI or in bulk using the BareMetalAsset CRD.

Once a user decides to create a cluster, they will use the ACM UI to select assets from the available BMAs and update the role (either "worker" or "master") and clusterDeployment association (name and namespace) for each BMA. Addition or removal of assets from the cluster happen in a similar way.

## Where does the inventory-controller live?

The inventory controller is packaged as part of controller operator. It can be enabled using the `--enable-inventory` flag.

```bash
./controller --enable-inventory
```
