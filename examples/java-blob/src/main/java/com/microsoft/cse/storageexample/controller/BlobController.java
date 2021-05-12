package com.microsoft.cse.storageexample.controller;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.nio.charset.StandardCharsets;

import javax.annotation.PostConstruct;

import com.azure.storage.blob.BlobClient;
import com.azure.storage.blob.BlobContainerClient;
import com.microsoft.cse.storageexample.config.BlobConfigLoader;

@RestController
@RequestMapping("/")
public class BlobController {

	@Autowired
	private BlobConfigLoader blobServiceClientBuilder;

	private BlobContainerClient containerClient;

	@PostConstruct
	private void init() {
		this.containerClient = blobServiceClientBuilder.client();
	}

	@GetMapping
	public String readBlobFile(@RequestParam String fileName) throws Exception {
		ByteArrayOutputStream stream = new ByteArrayOutputStream();

		BlobClient blobClient = containerClient.getBlobClient(fileName);
		blobClient.download(stream);

		return stream.toString();
	}

	@PostMapping
	public String writeBlobFile(@RequestBody String data) throws Exception {
		String fileName = String.format("quickstart-%s.txt", java.util.UUID.randomUUID());

		BlobClient blobClient = containerClient.getBlobClient(fileName);
		blobClient.upload(new ByteArrayInputStream(data.getBytes(StandardCharsets.UTF_8)), data.length());

		return String.format("file %s was uploaded", fileName);
	}
}
