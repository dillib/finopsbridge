"use client"

import { useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { apiRequestWithAuth } from "@/lib/api"
import { useAuth } from "@clerk/nextjs"
import { useToast } from "@/hooks/use-toast"
import { Textarea } from "@/components/ui/textarea"

const awsSchema = z.object({
  name: z.string().min(1, "Name is required"),
  roleArn: z.string().min(1, "Role ARN is required"),
  accountId: z.string().optional(),
})

const azureSchema = z.object({
  name: z.string().min(1, "Name is required"),
  subscriptionId: z.string().min(1, "Subscription ID is required"),
  servicePrincipalId: z.string().min(1, "Service Principal ID is required"),
  servicePrincipalSecret: z.string().min(1, "Service Principal Secret is required"),
  tenantId: z.string().min(1, "Tenant ID is required"),
})

const gcpSchema = z.object({
  name: z.string().min(1, "Name is required"),
  projectId: z.string().min(1, "Project ID is required"),
  serviceAccountKey: z.string().min(1, "Service Account JSON is required"),
})

const ociSchema = z.object({
  name: z.string().min(1, "Name is required"),
  tenancyId: z.string().min(1, "Tenancy OCID is required"),
  compartmentId: z.string().min(1, "Compartment OCID is required"),
  userId: z.string().min(1, "User OCID is required"),
  fingerprint: z.string().min(1, "API Key Fingerprint is required"),
  privateKey: z.string().min(1, "Private Key is required"),
  region: z.string().min(1, "Region is required"),
})

const ibmSchema = z.object({
  name: z.string().min(1, "Name is required"),
  accountId: z.string().min(1, "Account ID is required"),
  apiKey: z.string().min(1, "API Key is required"),
})

export default function SettingsPage() {
  const { getToken } = useAuth()
  const { toast } = useToast()
  const [activeTab, setActiveTab] = useState("aws")

  const awsForm = useForm<z.infer<typeof awsSchema>>({
    resolver: zodResolver(awsSchema),
  })

  const azureForm = useForm<z.infer<typeof azureSchema>>({
    resolver: zodResolver(azureSchema),
  })

  const gcpForm = useForm<z.infer<typeof gcpSchema>>({
    resolver: zodResolver(gcpSchema),
  })

  const ociForm = useForm<z.infer<typeof ociSchema>>({
    resolver: zodResolver(ociSchema),
  })

  const ibmForm = useForm<z.infer<typeof ibmSchema>>({
    resolver: zodResolver(ibmSchema),
  })

  const onSubmitAWS = async (data: z.infer<typeof awsSchema>) => {
    try {
      const token = await getToken()
      if (!token) throw new Error("Not authenticated")

      await apiRequestWithAuth("/api/cloud-providers", token, {
        method: "POST",
        body: JSON.stringify({
          type: "aws",
          ...data,
          credentials: {
            roleArn: data.roleArn,
          },
        }),
      })

      toast({
        title: "AWS connected",
        description: "Your AWS account has been connected successfully.",
      })

      awsForm.reset()
    } catch (error) {
      toast({
        title: "Error",
        description: error instanceof Error ? error.message : "Failed to connect AWS",
        variant: "destructive",
      })
    }
  }

  const onSubmitAzure = async (data: z.infer<typeof azureSchema>) => {
    try {
      const token = await getToken()
      if (!token) throw new Error("Not authenticated")

      await apiRequestWithAuth("/api/cloud-providers", token, {
        method: "POST",
        body: JSON.stringify({
          type: "azure",
          ...data,
          credentials: {
            servicePrincipalId: data.servicePrincipalId,
            servicePrincipalSecret: data.servicePrincipalSecret,
            tenantId: data.tenantId,
          },
        }),
      })

      toast({
        title: "Azure connected",
        description: "Your Azure subscription has been connected successfully.",
      })

      azureForm.reset()
    } catch (error) {
      toast({
        title: "Error",
        description: error instanceof Error ? error.message : "Failed to connect Azure",
        variant: "destructive",
      })
    }
  }

  const onSubmitGCP = async (data: z.infer<typeof gcpSchema>) => {
    try {
      const token = await getToken()
      if (!token) throw new Error("Not authenticated")

      await apiRequestWithAuth("/api/cloud-providers", token, {
        method: "POST",
        body: JSON.stringify({
          type: "gcp",
          ...data,
          credentials: {
            serviceAccountKey: data.serviceAccountKey,
          },
        }),
      })

      toast({
        title: "GCP connected",
        description: "Your GCP project has been connected successfully.",
      })

      gcpForm.reset()
    } catch (error) {
      toast({
        title: "Error",
        description: error instanceof Error ? error.message : "Failed to connect GCP",
        variant: "destructive",
      })
    }
  }

  const onSubmitOCI = async (data: z.infer<typeof ociSchema>) => {
    try {
      const token = await getToken()
      if (!token) throw new Error("Not authenticated")

      await apiRequestWithAuth("/api/cloud-providers", token, {
        method: "POST",
        body: JSON.stringify({
          type: "oci",
          name: data.name,
          tenancyId: data.tenancyId,
          compartmentId: data.compartmentId,
          credentials: {
            ociUserId: data.userId,
            ociFingerprint: data.fingerprint,
            ociPrivateKey: data.privateKey,
            region: data.region,
          },
        }),
      })

      toast({
        title: "OCI connected",
        description: "Your Oracle Cloud Infrastructure account has been connected successfully.",
      })

      ociForm.reset()
    } catch (error) {
      toast({
        title: "Error",
        description: error instanceof Error ? error.message : "Failed to connect OCI",
        variant: "destructive",
      })
    }
  }

  const onSubmitIBM = async (data: z.infer<typeof ibmSchema>) => {
    try {
      const token = await getToken()
      if (!token) throw new Error("Not authenticated")

      await apiRequestWithAuth("/api/cloud-providers", token, {
        method: "POST",
        body: JSON.stringify({
          type: "ibm",
          name: data.name,
          ibmAccountId: data.accountId,
          credentials: {
            ibmApiKey: data.apiKey,
          },
        }),
      })

      toast({
        title: "IBM Cloud connected",
        description: "Your IBM Cloud account has been connected successfully.",
      })

      ibmForm.reset()
    } catch (error) {
      toast({
        title: "Error",
        description: error instanceof Error ? error.message : "Failed to connect IBM Cloud",
        variant: "destructive",
      })
    }
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold">Settings</h1>
        <p className="text-muted-foreground">Manage your cloud provider connections</p>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="grid w-full grid-cols-5">
          <TabsTrigger value="aws">AWS</TabsTrigger>
          <TabsTrigger value="azure">Azure</TabsTrigger>
          <TabsTrigger value="gcp">GCP</TabsTrigger>
          <TabsTrigger value="oci">OCI</TabsTrigger>
          <TabsTrigger value="ibm">IBM</TabsTrigger>
        </TabsList>

        <TabsContent value="aws">
          <Card>
            <CardHeader>
              <CardTitle>Connect AWS</CardTitle>
              <CardDescription>
                Connect your AWS account using a cross-account IAM role
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={awsForm.handleSubmit(onSubmitAWS)} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="aws-name">Connection Name</Label>
                  <Input
                    id="aws-name"
                    {...awsForm.register("name")}
                    placeholder="Production AWS"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="aws-role-arn">IAM Role ARN</Label>
                  <Input
                    id="aws-role-arn"
                    {...awsForm.register("roleArn")}
                    placeholder="arn:aws:iam::123456789012:role/FinOpsBridgeRole"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="aws-account-id">Account ID (optional)</Label>
                  <Input
                    id="aws-account-id"
                    {...awsForm.register("accountId")}
                    placeholder="123456789012"
                  />
                </div>
                <Button type="submit">Connect AWS</Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="azure">
          <Card>
            <CardHeader>
              <CardTitle>Connect Azure</CardTitle>
              <CardDescription>
                Connect your Azure subscription using a service principal
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={azureForm.handleSubmit(onSubmitAzure)} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="azure-name">Connection Name</Label>
                  <Input
                    id="azure-name"
                    {...azureForm.register("name")}
                    placeholder="Production Azure"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="azure-subscription-id">Subscription ID</Label>
                  <Input
                    id="azure-subscription-id"
                    {...azureForm.register("subscriptionId")}
                    placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="azure-sp-id">Service Principal ID</Label>
                  <Input
                    id="azure-sp-id"
                    {...azureForm.register("servicePrincipalId")}
                    placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="azure-sp-secret">Service Principal Secret</Label>
                  <Input
                    id="azure-sp-secret"
                    type="password"
                    {...azureForm.register("servicePrincipalSecret")}
                    placeholder="Enter secret"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="azure-tenant-id">Tenant ID</Label>
                  <Input
                    id="azure-tenant-id"
                    {...azureForm.register("tenantId")}
                    placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
                  />
                </div>
                <Button type="submit">Connect Azure</Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="gcp">
          <Card>
            <CardHeader>
              <CardTitle>Connect GCP</CardTitle>
              <CardDescription>
                Connect your GCP project using a service account JSON key
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={gcpForm.handleSubmit(onSubmitGCP)} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="gcp-name">Connection Name</Label>
                  <Input
                    id="gcp-name"
                    {...gcpForm.register("name")}
                    placeholder="Production GCP"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="gcp-project-id">Project ID</Label>
                  <Input
                    id="gcp-project-id"
                    {...gcpForm.register("projectId")}
                    placeholder="my-gcp-project"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="gcp-sa-key">Service Account JSON Key</Label>
                  <Textarea
                    id="gcp-sa-key"
                    {...gcpForm.register("serviceAccountKey")}
                    placeholder="Paste your service account JSON key here"
                    rows={10}
                  />
                </div>
                <Button type="submit">Connect GCP</Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="oci">
          <Card>
            <CardHeader>
              <CardTitle>Connect Oracle Cloud Infrastructure</CardTitle>
              <CardDescription>
                Connect your OCI tenancy using API key authentication
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={ociForm.handleSubmit(onSubmitOCI)} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="oci-name">Connection Name</Label>
                  <Input
                    id="oci-name"
                    {...ociForm.register("name")}
                    placeholder="Production OCI"
                  />
                </div>
                <div className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <Label htmlFor="oci-tenancy-id">Tenancy OCID</Label>
                    <Input
                      id="oci-tenancy-id"
                      {...ociForm.register("tenancyId")}
                      placeholder="ocid1.tenancy.oc1..aaa..."
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="oci-compartment-id">Compartment OCID</Label>
                    <Input
                      id="oci-compartment-id"
                      {...ociForm.register("compartmentId")}
                      placeholder="ocid1.compartment.oc1..aaa..."
                    />
                  </div>
                </div>
                <div className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <Label htmlFor="oci-user-id">User OCID</Label>
                    <Input
                      id="oci-user-id"
                      {...ociForm.register("userId")}
                      placeholder="ocid1.user.oc1..aaa..."
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="oci-region">Region</Label>
                    <Input
                      id="oci-region"
                      {...ociForm.register("region")}
                      placeholder="us-ashburn-1"
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="oci-fingerprint">API Key Fingerprint</Label>
                  <Input
                    id="oci-fingerprint"
                    {...ociForm.register("fingerprint")}
                    placeholder="aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="oci-private-key">Private Key (PEM)</Label>
                  <Textarea
                    id="oci-private-key"
                    {...ociForm.register("privateKey")}
                    placeholder="-----BEGIN RSA PRIVATE KEY-----&#10;...&#10;-----END RSA PRIVATE KEY-----"
                    rows={6}
                  />
                </div>
                <Button type="submit">Connect OCI</Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="ibm">
          <Card>
            <CardHeader>
              <CardTitle>Connect IBM Cloud</CardTitle>
              <CardDescription>
                Connect your IBM Cloud account using an API key
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={ibmForm.handleSubmit(onSubmitIBM)} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="ibm-name">Connection Name</Label>
                  <Input
                    id="ibm-name"
                    {...ibmForm.register("name")}
                    placeholder="Production IBM Cloud"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="ibm-account-id">Account ID</Label>
                  <Input
                    id="ibm-account-id"
                    {...ibmForm.register("accountId")}
                    placeholder="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="ibm-api-key">API Key</Label>
                  <Input
                    id="ibm-api-key"
                    type="password"
                    {...ibmForm.register("apiKey")}
                    placeholder="Enter your IBM Cloud API key"
                  />
                  <p className="text-xs text-muted-foreground">
                    Create an API key in IBM Cloud IAM with Viewer access to billing and VPC services
                  </p>
                </div>
                <Button type="submit">Connect IBM Cloud</Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}

