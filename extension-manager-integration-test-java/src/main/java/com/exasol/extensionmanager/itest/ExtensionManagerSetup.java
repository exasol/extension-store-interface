package com.exasol.extensionmanager.itest;

import java.io.*;
import java.nio.file.Files;
import java.nio.file.Path;
import java.sql.*;
import java.util.*;
import java.util.logging.Logger;
import java.util.stream.Stream;

import com.exasol.dbbuilder.dialects.exasol.ExasolObjectFactory;
import com.exasol.dbbuilder.dialects.exasol.ExasolSchema;
import com.exasol.errorreporting.ExaError;
import com.exasol.exasoltestsetup.ExasolTestSetup;
import com.exasol.extensionmanager.itest.builder.ExtensionBuilder;
import com.exasol.extensionmanager.itest.installer.ExtensionManagerInstaller;

/**
 * Main class responsible for setting up the environment required for testing extensions using the extension manager.
 */
public class ExtensionManagerSetup implements AutoCloseable {
    private static final Logger LOGGER = Logger.getLogger(ExtensionManagerSetup.class.getName());
    private static final String EXTENSION_SCHEMA_NAME = "EXA_EXTENSIONS";
    private final ExtensionManagerProcess extensionManager;
    private final ExasolTestSetup exasolTestSetup;
    private final ExasolObjectFactory exasolObjectFactory;
    private final Connection connection;
    private final List<Runnable> cleanupCallbacks = new ArrayList<>();
    private final ExtensionManagerClient client;
    private final Path tempDir;

    private ExtensionManagerSetup(final ExtensionManagerProcess extensionManager, final ExasolTestSetup exasolTestSetup,
            final ExasolObjectFactory exasolObjectFactory, final ExtensionManagerClient client, final Path tempDir) {
        this.extensionManager = extensionManager;
        this.exasolTestSetup = exasolTestSetup;
        this.exasolObjectFactory = exasolObjectFactory;
        this.client = client;
        this.tempDir = tempDir;
        try {
            this.connection = this.exasolTestSetup.createConnection();
        } catch (final SQLException exception) {
            throw new AssertionError("Failed to create db connection", exception);
        }
    }

    /**
     * Prepare and create a new instance of {@link ExtensionManagerSetup}. Usually you call this in a
     * {@link org.junit.jupiter.api.BeforeAll} method. Make sure to close this by calling {@link #close()} in an
     * {@link org.junit.jupiter.api.AfterAll} method.
     * 
     * @param exasolTestSetup     exasol test setup to use for the tests
     * @param exasolObjectFactory object factory for creating exasol objects
     * @param extensionBuilder    builder for building the extension under test
     * @return a new instance
     */
    public static ExtensionManagerSetup create(final ExasolTestSetup exasolTestSetup,
            final ExasolObjectFactory exasolObjectFactory, final ExtensionBuilder extensionBuilder) {
        final Path tempDir = createTempDir();
        final ExtensionTestConfig config = ExtensionTestConfig.read();
        prepareExtension(config, extensionBuilder, tempDir);
        final ExtensionManagerInstaller installer = ExtensionManagerInstaller.forConfig(config);
        final Path extensionManagerExecutable = installer.install();
        final ExtensionManagerProcess extensionManager = ExtensionManagerProcess.start(extensionManagerExecutable,
                tempDir);
        final ExtensionManagerClient client = ExtensionManagerClient.create(extensionManager.getServerBasePath(),
                exasolTestSetup.getConnectionInfo());
        return new ExtensionManagerSetup(extensionManager, exasolTestSetup, exasolObjectFactory, client, tempDir);
    }

    private static Path createTempDir() {
        try {
            return Files.createTempDirectory("extension-manager-itest");
        } catch (final IOException exception) {
            throw new UncheckedIOException("Failed to create temp directory", exception);
        }
    }

    private static void prepareExtension(final ExtensionTestConfig config, final ExtensionBuilder extensionBuilder,
            final Path extensionRegistryDir) {
        if (config.buildExtension()) {
            extensionBuilder.build();
        } else {
            LOGGER.warning("Skip building extension");
        }
        final Path extensionFile = extensionBuilder.getExtensionFile();
        if (!Files.exists(extensionFile)) {
            throw new IllegalStateException(ExaError.messageBuilder("E-EMIT-1")
                    .message("Extension file {{extension file}} not found.", extensionFile)
                    .mitigation("Set buildExtension to true in {{config file}}.", config.getConfigFile())
                    .mitigation("Ensure that extension was built successfully.").toString());
        }
        LOGGER.info(() -> "Extension " + extensionFile + " was built successfully");
        copy(extensionFile, extensionRegistryDir.resolve(extensionFile.getFileName()));
    }

    private static void copy(final Path sourceFile, final Path targetFile) {
        try {
            Files.copy(sourceFile, targetFile);
        } catch (final IOException exception) {
            throw new UncheckedIOException("Error copying extension " + sourceFile + " to " + targetFile, exception);
        }
    }

    /**
     * Get the client for accessing the extension manager via its REST API. Use this for calling and testing methods of
     * the extension under test.
     * 
     * @return extension manager client
     */
    public ExtensionManagerClient client() {
        return this.client;
    }

    /**
     * Get access to Exasol's metadata tables. This is useful for verifying that the extension under test created
     * expected objects like {@code SCRIPT}s or {@code CONNECTION}s.
     * 
     * @return exasol metadata
     */
    public ExasolMetadata exasolMetadata() {
        return new ExasolMetadata(this.connection, EXTENSION_SCHEMA_NAME);
    }

    /**
     * Create the extension schema used by extension manager. This is useful for testing if the extension under test can
     * handle existing database objects.
     * 
     * @return new extension schema.
     */
    public ExasolSchema createExtensionSchema() {
        return this.exasolObjectFactory.createSchema(EXTENSION_SCHEMA_NAME);
    }

    /**
     * Drop the virtual schema with the given name when calling {@link #close()}.
     * 
     * @param name the virtual schema to drop
     */
    public void addVirtualSchemaToCleanupQueue(final String name) {
        this.cleanupCallbacks.add(dropVirtualSchema(name));
    }

    private Runnable dropVirtualSchema(final String name) {
        return () -> {
            try {
                createStatement().execute("DROP VIRTUAL SCHEMA IF EXISTS \"" + name + "\" CASCADE");
            } catch (final SQLException exception) {
                throw new IllegalStateException("Failed to drop virtual schema " + name, exception);
            }
        };
    }

    /**
     * Drop the connection with the given name when calling {@link #close()}.
     * 
     * @param name the connection to drop
     */
    public void addConnectionToCleanupQueue(final String name) {
        this.cleanupCallbacks.add(dropConnection(name));
    }

    private Runnable dropConnection(final String name) {
        return () -> {
            try {
                createStatement().execute("DROP CONNECTION IF EXISTS \"" + name + "\"");
            } catch (final SQLException exception) {
                throw new IllegalStateException("Failed to drop connection " + name, exception);
            }
        };
    }

    private Statement createStatement() throws SQLException {
        return this.connection.createStatement();
    }

    /**
     * Cleanup resources after running tests. Call this in a {@link org.junit.jupiter.api.AfterAll} method.
     */
    @Override
    public void close() {
        dropExtensionSchema();
        deleteTempDir();
        extensionManager.close();
        try {
            this.exasolTestSetup.close();
        } catch (final Exception exception) {
            throw new IllegalStateException("Error closing exasol test setup", exception);
        }
    }

    private void deleteTempDir() {
        try (Stream<Path> files = Files.walk(this.tempDir)) {
            files.sorted(Comparator.reverseOrder()) //
                    .map(Path::toFile) //
                    .forEach(File::delete);
        } catch (final IOException exception) {
            throw new UncheckedIOException("Failed to delete temp dir " + tempDir, exception);
        }
    }

    private void dropExtensionSchema() {
        this.extensionManager.close();
        this.cleanupCallbacks.forEach(Runnable::run);
        this.cleanupCallbacks.clear();
        try {
            createStatement().execute("DROP SCHEMA IF EXISTS \"" + EXTENSION_SCHEMA_NAME + "\" CASCADE");
        } catch (final SQLException exception) {
            throw new IllegalStateException("Failed to delete extension schema " + EXTENSION_SCHEMA_NAME, exception);
        }
    }
}
