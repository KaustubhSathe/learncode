package main.java;

import com.amazonaws.services.lambda.runtime.Context;
import com.amazonaws.services.lambda.runtime.RequestHandler;
import com.amazonaws.services.lambda.runtime.events.APIGatewayProxyRequestEvent;
import com.amazonaws.services.lambda.runtime.events.APIGatewayProxyResponseEvent;
import com.google.gson.Gson;
import software.amazon.awssdk.services.dynamodb.DynamoDbClient;
import software.amazon.awssdk.services.dynamodb.model.AttributeValue;
import software.amazon.awssdk.services.dynamodb.model.UpdateItemRequest;
import software.amazon.awssdk.services.dynamodb.model.GetItemRequest;
import software.amazon.awssdk.services.dynamodb.model.GetItemResponse;

import javax.tools.*;
import java.util.Base64;
import java.util.HashMap;
import java.util.Map;
import java.io.*;
import java.nio.file.*;
import java.util.Arrays;
import java.util.Scanner;

public class Handler implements RequestHandler<APIGatewayProxyRequestEvent, APIGatewayProxyResponseEvent> {
    private static final Gson gson = new Gson();
    private static final DynamoDbClient dynamoDB = DynamoDbClient.builder().build();
    private static final JavaCompiler compiler = ToolProvider.getSystemJavaCompiler();
    
    public APIGatewayProxyResponseEvent handleRequest(APIGatewayProxyRequestEvent event, Context context) {
        context.getLogger().log("Processing Java runner request");
        context.getLogger().log("Request path: " + event.getPath());
        context.getLogger().log("Request method: " + event.getHttpMethod());
        
        Submission submission = null;
        try {
            // Parse the request
            context.getLogger().log("Parsing request body");
            if (event.getBody() == null) {
                throw new RuntimeException("Request body is empty");
            }
            RunRequest runRequest = gson.fromJson(event.getBody(), RunRequest.class);
            context.getLogger().log("Request parsed - Cache: " + runRequest.cache + ", Topic: " + runRequest.topic);
            
            // Decode the base64 submission
            context.getLogger().log("Decoding base64 submission");
            if (runRequest.binary == null) {
                throw new RuntimeException("Binary submission data is missing");
            }
            byte[] decodedBytes = Base64.getDecoder().decode(runRequest.binary);
            String submissionJson = new String(decodedBytes);
            context.getLogger().log("Decoded submission: " + submissionJson);
            
            // Parse the submission
            submission = gson.fromJson(submissionJson, Submission.class);
            context.getLogger().log("Submission details - ID: " + submission.id + 
                ", Problem ID: " + submission.problem_id + 
                ", User ID: " + submission.user_id +
                ", Language: " + submission.language +
                ", Code length: " + (submission.code != null ? submission.code.length() : 0));
            
            // Update submission status to running
            context.getLogger().log("Updating submission status to running");
            updateSubmissionStatus(submission.id, "running", null, context, submission);
            
            // Execute the Java code
            context.getLogger().log("Executing Java code");
            String result = executeJava(submission.code, submission.problem_id, context);
            context.getLogger().log("Execution result: " + result);
            
            // Update final status
            context.getLogger().log("Updating submission status to completed");
            updateSubmissionStatus(submission.id, "completed", result, context, submission);
            
            context.getLogger().log("Request completed successfully");
            return new APIGatewayProxyResponseEvent()
                .withStatusCode(200)
                .withBody(gson.toJson(Map.of("result", result)));
                
        } catch (Exception e) {
            context.getLogger().log("Request failed with error: " + e.getClass().getName());
            context.getLogger().log("Error: " + e.getMessage() + "\n" + Arrays.toString(e.getStackTrace()));
            String errorMessage = e.getMessage() != null ? e.getMessage() : "Unknown error occurred";
            
            // Update submission status to error
            try {
                context.getLogger().log("Updating submission status to error");
                submission.result = errorMessage;
                updateSubmissionStatus(submission.id, "error", null, context, submission);
            } catch (Exception updateErr) {
                context.getLogger().log("Failed to update error status: " + updateErr.getMessage());
            }
            
            return new APIGatewayProxyResponseEvent()
                .withStatusCode(500)
                .withBody(gson.toJson(Map.of("error", errorMessage)));
        }
    }
    
    private void updateSubmissionStatus(String id, String status, String result, Context context, Submission submission) {
        context.getLogger().log("Updating submission status - ID: " + id + ", Status: " + status);
        Map<String, AttributeValue> key = new HashMap<>();
        key.put("id", AttributeValue.builder().s(id).build());
        
        Map<String, AttributeValue> values = new HashMap<>();
        values.put(":s", AttributeValue.builder().s(status).build());
        
        context.getLogger().log("Building DynamoDB update request");
        UpdateItemRequest.Builder updateBuilder = UpdateItemRequest.builder()
            .tableName(System.getenv("SUBMISSIONS_TABLE"))
            .key(key)
            .updateExpression("SET #status = :s")
            .expressionAttributeNames(Map.of("#status", "status"));
            
        if (status.equals("error")) {
            context.getLogger().log("Including error message in update");
            values.put(":e", AttributeValue.builder().s(submission.result).build());
            updateBuilder.updateExpression("SET #status = :s, #result = :e")
                .expressionAttributeNames(Map.of(
                    "#status", "status",
                    "#result", "result"
                ));
        } else if (result != null) {
            context.getLogger().log("Including result in update, length: " + result.length());
            values.put(":r", AttributeValue.builder().s(result).build());
            updateBuilder.updateExpression("SET #status = :s, #result = :r")
                .expressionAttributeNames(Map.of(
                    "#status", "status",
                    "#result", "result"
                ));
        }
        
        context.getLogger().log("Executing DynamoDB update");
        try {
            dynamoDB.updateItem(updateBuilder.expressionAttributeValues(values).build());
            context.getLogger().log("Update successful");
        } catch (Exception e) {
            context.getLogger().log("Update failed: " + e.getMessage());
            throw e;
        }
    }
    
    private Problem getProblem(String problemId, Context context) {
        context.getLogger().log("Getting problem details for ID: " + problemId);
        Map<String, AttributeValue> key = new HashMap<>();
        key.put("id", AttributeValue.builder().s(problemId).build());
        
        context.getLogger().log("Querying DynamoDB table: " + System.getenv("PROBLEMS_TABLE"));
        GetItemResponse response = dynamoDB.getItem(GetItemRequest.builder()
            .tableName(System.getenv("PROBLEMS_TABLE"))
            .key(key)
            .build());
            
        if (!response.hasItem()) {
            context.getLogger().log("Problem not found in DynamoDB");
            throw new RuntimeException("Problem not found: " + problemId);
        }
        
        context.getLogger().log("Problem found, extracting attributes");
        Map<String, AttributeValue> item = response.item();
        AttributeValue input = item.get("input");
        AttributeValue output = item.get("output");
        
        if (input == null || output == null) {
            context.getLogger().log("Missing attributes - input: " + (input == null) + ", output: " + (output == null));
            throw new RuntimeException("Problem is missing input/output");
        }
        
        context.getLogger().log("Successfully retrieved problem details");
        return new Problem(
            input.s(),
            output.s()
        );
    }
    
    private String executeJava(String code, String problemId, Context context) throws Exception {
        context.getLogger().log("Starting Java code execution");
        // Create temporary directory
        Path tempDir = Files.createTempDirectory("java-execution");
        context.getLogger().log("Created temp directory: " + tempDir);
        
        // Get problem details
        Problem problem = getProblem(problemId, context);
        context.getLogger().log("Retrieved problem details with input length: " + problem.input.length());
        
        try {
            // Create Solution.java file
            Path sourcePath = tempDir.resolve("Solution.java");
            context.getLogger().log("Writing code to: " + sourcePath);
            Files.write(sourcePath, code.getBytes());
            
            // Compile the code
            context.getLogger().log("Starting compilation");
            DiagnosticCollector<JavaFileObject> diagnostics = new DiagnosticCollector<>();
            StandardJavaFileManager fileManager = compiler.getStandardFileManager(diagnostics, null, null);
            Iterable<? extends JavaFileObject> compilationUnits = fileManager.getJavaFileObjectsFromFiles(Arrays.asList(sourcePath.toFile()));
            
            JavaCompiler.CompilationTask task = compiler.getTask(
                null,
                fileManager,
                diagnostics,
                null,
                null,
                compilationUnits);
                
            if (!task.call()) {
                context.getLogger().log("Compilation failed");
                StringBuilder errors = new StringBuilder("Compilation failed:\n");
                for (Diagnostic<?> diagnostic : diagnostics.getDiagnostics()) {
                    errors.append(diagnostic.getMessage(null)).append("\n");
                }
                throw new Exception(errors.toString());
            }
            
            context.getLogger().log("Compilation successful, running code");
            // Run the compiled code
            ProcessBuilder pb = new ProcessBuilder("java", "-cp", tempDir.toString(), "Solution");
            pb.directory(tempDir.toFile());
            Process process = pb.start();
            
            // Provide input
            context.getLogger().log("Writing input to process");
            try (OutputStreamWriter writer = new OutputStreamWriter(process.getOutputStream())) {
                writer.write(problem.input);
                writer.flush();
            }
            
            // Get output
            context.getLogger().log("Reading process output");
            String output = new String(process.getInputStream().readAllBytes()).trim();
            String expectedOutput = problem.output.trim();
            
            // Check if output matches
            context.getLogger().log("Comparing output - Expected length: " + expectedOutput.length() + ", Got length: " + output.length());
            if (!output.equals(expectedOutput)) {
                context.getLogger().log("Output mismatch detected");
                throw new Exception(String.format(
                    "Output mismatch!\nExpected:\n%s\nGot:\n%s",
                    expectedOutput,
                    output
                ));
            }
            
            context.getLogger().log("Code execution successful");
            return output;
            
        } finally {
            // Cleanup
            context.getLogger().log("Cleaning up temporary files");
            Files.walk(tempDir)
                .sorted((a, b) -> b.compareTo(a))
                .forEach(path -> {
                    try {
                        Files.delete(path);
                    } catch (IOException e) {
                        context.getLogger().log("Failed to delete: " + path + " - " + e.getMessage());
                    }
                });
        }
    }
}

class Problem {
    public final String input;
    public final String output;
    
    public Problem(String input, String output) {
        this.input = input;
        this.output = output;
    }
}

class RunRequest {
    public String cache;
    public String topic;
    public String binary;
}

class Submission {
    public String id;
    public String user_id;
    public String problem_id;
    public String language;
    public String code;
    public String status;
    public String result;
    public long created_at;
    public long updated_at;
} 