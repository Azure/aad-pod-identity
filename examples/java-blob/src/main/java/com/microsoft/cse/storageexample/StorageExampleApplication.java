package com.microsoft.cse.storageexample;

import org.slf4j.LoggerFactory;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

@SpringBootApplication
public class StorageExampleApplication {
	org.slf4j.Logger logger = LoggerFactory.getLogger(StorageExampleApplication.class);

	public static void main(String[] args) {
		SpringApplication.run(StorageExampleApplication.class, args);
	}
}
