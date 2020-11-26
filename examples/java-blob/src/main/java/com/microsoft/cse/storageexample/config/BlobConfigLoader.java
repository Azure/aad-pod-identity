package com.microsoft.cse.storageexample.config;

import java.util.Locale;

import com.azure.identity.DefaultAzureCredential;
import com.azure.identity.DefaultAzureCredentialBuilder;
import com.azure.storage.blob.BlobContainerClient;
import com.azure.storage.blob.BlobServiceClient;
import com.azure.storage.blob.BlobServiceClientBuilder;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

@Component("blobConfigLoader")
public class BlobConfigLoader {
	private static Logger logger = LoggerFactory.getLogger(BlobConfigLoader.class);

	public BlobContainerClient client() {
		String msiClientId = System.getenv("AZURE_CLIENT_ID");
		logger.info("Manage Identity {}", msiClientId);

		String accountName = System.getenv("BLOB_ACCOUNT_NAME");
		logger.info("Blob Storage Name {}", accountName);

		String containerName = System.getenv("BLOB_CONTAINER_NAME");
		logger.info("Blob Storage Container {}", containerName);

		DefaultAzureCredential defaultCredential = new DefaultAzureCredentialBuilder()
				.managedIdentityClientId(msiClientId)
				.build();

		String endpoint = String.format(Locale.ROOT, "https://%s.blob.core.windows.net", accountName);

		/*
		 * Create a storage client using the Azure Identity credentials.
		 */
		BlobServiceClient storageClient = new BlobServiceClientBuilder()
				.endpoint(endpoint)
				.credential(defaultCredential)
				.buildClient();

		BlobContainerClient containerClient = storageClient.getBlobContainerClient(containerName);

		return containerClient;
	}
}
