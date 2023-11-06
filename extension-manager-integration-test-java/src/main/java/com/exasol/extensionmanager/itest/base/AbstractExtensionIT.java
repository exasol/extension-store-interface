package com.exasol.extensionmanager.itest.base;

import static com.exasol.matcher.ResultSetStructureMatcher.table;
import static java.util.Collections.emptyList;
import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.*;
import static org.junit.jupiter.api.Assertions.assertAll;
import static org.junit.jupiter.api.Assertions.assertDoesNotThrow;

import java.util.*;
import java.util.logging.Logger;

import org.junit.jupiter.api.*;

import com.exasol.extensionmanager.client.model.*;
import com.exasol.extensionmanager.itest.*;

/**
 * This is a base class for Extension integration tests that already contains some basic tests for
 * installing/listing/uninstalling extensions and creating/listing/deleting instances.
 */
public abstract class AbstractExtensionIT {
    private static final String VIRTUAL_SCHEMA_NAME_PARAM_NAME = "base-vs.virtual-schema-name";
    private static final String EXTENSION_SCHEMA = "EXA_EXTENSIONS";
    private static final Logger LOG = Logger.getLogger(AbstractExtensionIT.class.getName());
    private final ExtensionITConfig config;

    /**
     * Create a new base integration test.
     */
    protected AbstractExtensionIT() {
        this.config = createConfig();
    }

    /**
     * Creates a new configuration for the integration tests.
     * 
     * @return new configuration
     */
    protected abstract ExtensionITConfig createConfig();

    /**
     * Get the {@link ExtensionManagerSetup extension manager setup}.
     * 
     * @return extension manager setup
     */
    protected abstract ExtensionManagerSetup getSetup();

    /**
     * Assert that the expected {@code SCRIPT}s exist after installing the extension.
     */
    protected abstract void assertScriptsExist();

    /**
     * Prepare test data for creating a new virtual schema using this extension. This contains e.g. creating a table in
     * the source schema or uploading test files to a cloud storage.
     */
    protected abstract void prepareInstance();

    /**
     * Assert that a newly created virtual schema contains the expected data.
     * 
     * @param virtualSchemaName name of the virtual schema to check
     */
    protected abstract void assertVirtualSchemaContent(final String virtualSchemaName);

    /**
     * Create the same {@code SCRIPT}s as the extension would do. This is used to check that the extension also detects
     * manually created scripts.
     */
    protected abstract void createScripts();

    /**
     * Create valid parameters for a new instance. The instance/virtual schema name will be added automatically and is
     * not required here.
     *
     * @return valid parameters for a new instance
     */
    protected abstract Collection<ParameterValue> createValidParameterValues();

    @BeforeEach
    void logTestName(final TestInfo testInfo) {
        LOG.info(">>> " + testInfo.getDisplayName());
    }

    @AfterEach
    void cleanup() {
        getSetup().cleanup();
    }

    @Test
    public void listExtensions() {
        final List<ExtensionsResponseExtension> extensions = getSetup().client().getExtensions();
        assertAll(() -> assertThat(extensions, hasSize(1)), //
                () -> assertThat(extensions.get(0).getName(), equalTo(config.getExtensionName())),
                () -> assertThat(extensions.get(0).getInstallableVersions().get(0).getName(),
                        equalTo(config.getCurrentVersion())),
                () -> assertThat(extensions.get(0).getInstallableVersions().get(0).isLatest(), is(true)),
                () -> assertThat(extensions.get(0).getInstallableVersions().get(0).isDeprecated(), is(false)),
                () -> assertThat(extensions.get(0).getDescription(), equalTo(config.getExtensionDescription())));
    }

    @Test
    public void listInstallationsEmpty() {
        final List<InstallationsResponseInstallation> installations = getSetup().client().getInstallations();
        assertThat(installations, hasSize(0));
    }

    @Test
    public void listInstallationsFindsMatchingScripts() {
        createScripts();
        final List<InstallationsResponseInstallation> installations = getSetup().client().getInstallations();
        assertAll(() -> assertThat(installations, hasSize(1)), //
                () -> assertThat(installations.get(0).getName(), equalTo(config.getExtensionName())),
                () -> assertThat(installations.get(0).getVersion(), equalTo(config.getCurrentVersion())));
    }

    @Test
    public void listInstallationsFindsOwnInstallation() {
        getSetup().client().install();
        final List<InstallationsResponseInstallation> installations = getSetup().client().getInstallations();
        assertAll(() -> assertThat(installations, hasSize(1)), //
                () -> assertThat(installations.get(0).getName(), equalTo(config.getExtensionName())),
                () -> assertThat(installations.get(0).getVersion(), equalTo(config.getCurrentVersion())));
    }

    @Test
    public void getExtensionDetailsFailsForUnknownVersion() {
        getSetup().client().assertRequestFails(() -> getSetup().client().getExtensionDetails("unknownVersion"),
                equalTo("Version 'unknownVersion' not supported, can only use '" + config.getCurrentVersion() + "'."),
                equalTo(404));
    }

    @Test
    public void getExtensionDetailsSuccess() {
        final ExtensionDetailsResponse extensionDetails = getSetup().client()
                .getExtensionDetails(config.getCurrentVersion());
        final List<ParamDefinition> parameters = extensionDetails.getParameterDefinitions();
        final ParamDefinition param1 = new ParamDefinition().id(VIRTUAL_SCHEMA_NAME_PARAM_NAME)
                .name("Virtual Schema name").definition(Map.of( //
                        "id", VIRTUAL_SCHEMA_NAME_PARAM_NAME, //
                        "name", "Virtual Schema name", //
                        "description", "Name for the new virtual schema", //
                        "placeholder", "MY_VIRTUAL_SCHEMA", //
                        "regex", "[a-zA-Z_]+", //
                        "required", true, //
                        "type", "string"));
        assertAll(() -> assertThat(extensionDetails.getId(), equalTo(config.getExtensionId())),
                () -> assertThat(extensionDetails.getVersion(), equalTo(config.getCurrentVersion())),
                () -> assertThat(parameters, hasSize(config.getExpectedParameterCount())),
                () -> assertThat(parameters.get(0), equalTo(param1)));
    }

    @Test
    public void installCreatesScripts() {
        getSetup().client().install();
        assertScriptsExist();
    }

    @Test
    public void installWorksIfCalledTwice() {
        getSetup().client().install();
        getSetup().client().install();
        assertScriptsExist();
    }

    @Test
    public void createInstanceFailsWithoutRequiredParameters() {
        final ExtensionManagerClient client = getSetup().client();
        client.install();
        client.assertRequestFails(() -> client.createInstance(emptyList()), startsWith(
                "invalid parameters: Failed to validate parameter 'Virtual Schema name' (base-vs.virtual-schema-name): This is a required parameter."),
                equalTo(400));
    }

    @Test
    public void uninstallSucceedsForNonExistingInstallation() {
        assertDoesNotThrow(() -> getSetup().client().uninstall());
    }

    @Test
    public void uninstallRemovesAdapters() {
        getSetup().client().install();
        assertAll(this::assertScriptsExist, //
                () -> assertThat(getSetup().client().getInstallations(), hasSize(1)));
        getSetup().client().uninstall(config.getCurrentVersion());
        assertAll(() -> assertThat(getSetup().client().getInstallations(), is(empty())),
                () -> getSetup().exasolMetadata().assertNoScripts());
    }

    @Test
    public void upgradeFailsWhenNotInstalled() {
        getSetup().client().assertRequestFails(() -> getSetup().client().upgrade(), //
                allOf(startsWith("Not all required scripts are installed: Validation failed: Script"),
                        endsWith("is missing")),
                equalTo(412));
    }

    @Test
    public void upgradeFailsWhenAlreadyUpToDate() {
        getSetup().client().install();
        getSetup().client().assertRequestFails(() -> getSetup().client().upgrade(),
                "Extension is already installed in latest version " + config.getCurrentVersion(), 412);
    }

    @Test
    public void upgradeFromPreviousVersion() {
        final PreviousExtensionVersion previousVersion = createPreviousVersion();
        previousVersion.prepare();
        previousVersion.install();
        prepareInstance();
        final String virtualSchemaName = "my_upgrading_VS";
        createInstance(previousVersion.getExtensionId(), config.getPreviousVersion(), virtualSchemaName);
        assertVirtualSchemaContent(virtualSchemaName);
        assertInstalledVersion(config.getPreviousVersion(), previousVersion);
        previousVersion.upgrade();
        assertInstalledVersion(config.getCurrentVersion(), previousVersion);
        assertVirtualSchemaContent(virtualSchemaName);
    }

    private PreviousExtensionVersion createPreviousVersion() {
        return getSetup().previousVersionManager().newVersion().currentVersion(config.getCurrentVersion()) //
                .previousVersion(config.getPreviousVersion()) //
                .adapterFileName(config.getPreviousVersionJarFile()) //
                .extensionFileName(config.getExtensionId()) //
                .project(config.getProjectName()) //
                .build();
    }

    private void assertInstalledVersion(final String expectedVersion, final PreviousExtensionVersion previousVersion) {
        // The extension is installed twice (previous and current version), so each one returns one installation.
        assertThat(getSetup().client().getInstallations(),
                containsInAnyOrder(
                        new InstallationsResponseInstallation().name(config.getExtensionName()).version(expectedVersion)
                                .id(config.getExtensionId()), //
                        new InstallationsResponseInstallation().name(config.getExtensionName()).version(expectedVersion)
                                .id(previousVersion.getExtensionId())));
    }

    @Test
    public void virtualSchemaWorks() {
        getSetup().client().install();
        prepareInstance();
        createInstance("my_VS");
        assertVirtualSchemaContent("my_VS");
    }

    @Test
    public void listingInstancesNoVSExists() {
        assertThat(getSetup().client().listInstances(), hasSize(0));
    }

    @Test
    public void listInstances() {
        getSetup().client().install();
        final String name = "my_virtual_SCHEMA1";
        createInstance(name);
        assertThat(getSetup().client().listInstances(config.getCurrentVersion()),
                allOf(hasSize(1), equalTo(List.of(new Instance().id(name).name(name)))));
    }

    @Test
    public void createInstanceCreatesDbObjects() {
        getSetup().client().install();
        final String name = "my_virtual_SCHEMA";
        createInstance(name);

        getSetup().exasolMetadata()
                .assertConnection(table().row("MY_VIRTUAL_SCHEMA_CONNECTION", getInstanceComment(name)).matches());
        getSetup().exasolMetadata().assertVirtualSchema(table()
                .row("my_virtual_SCHEMA", "SYS", EXTENSION_SCHEMA, not(emptyOrNullString()), not(emptyOrNullString()))
                .matches());
        assertThat(getSetup().client().listInstances(),
                allOf(hasSize(1), equalTo(List.of(new Instance().id(name).name(name)))));
    }

    private String getInstanceComment(final String instanceName) {
        return "Created by Extension Manager for " + config.getExtensionName() + " v" + config.getCurrentVersion() + " "
                + instanceName;
    }

    @Test
    public void createTwoInstances() {
        getSetup().client().install();
        createInstance("vs1");
        createInstance("vs2");

        assertAll(
                () -> getSetup().exasolMetadata()
                        .assertConnection(table().row("VS1_CONNECTION", getInstanceComment("vs1"))
                                .row("VS2_CONNECTION", getInstanceComment("vs2")).matches()),
                () -> getSetup().exasolMetadata()
                        .assertVirtualSchema(table()
                                .row("vs1", "SYS", EXTENSION_SCHEMA, not(emptyOrNullString()), not(emptyOrNullString()))
                                .row("vs2", "SYS", EXTENSION_SCHEMA, not(emptyOrNullString()), not(emptyOrNullString()))
                                .matches()),

                () -> assertThat(getSetup().client().listInstances(), allOf(hasSize(2),
                        equalTo(List.of(new Instance().id("vs1").name("vs1"), new Instance().id("vs2").name("vs2"))))));
    }

    @Test
    public void createInstanceWithSingleQuote() {
        getSetup().client().install();
        createInstance("Quoted'schema");
        assertAll(
                () -> getSetup().exasolMetadata().assertConnection(
                        table().row("QUOTED'SCHEMA_CONNECTION", getInstanceComment("Quoted'schema")).matches()),
                () -> getSetup().exasolMetadata().assertVirtualSchema(table()
                        .row("Quoted'schema", "SYS", EXTENSION_SCHEMA, "S3_FILES_ADAPTER", not(emptyOrNullString()))
                        .matches()));
    }

    @Test
    public void deleteNonExistingInstance() {
        assertDoesNotThrow(() -> getSetup().client().deleteInstance("no-such-instance"));
    }

    @Test
    public void deleteFailsForUnknownVersion() {
        getSetup().client().assertRequestFails(
                () -> getSetup().client().deleteInstance("unknownVersion", "no-such-instance"),
                equalTo("Version 'unknownVersion' not supported, can only use '" + config.getCurrentVersion() + "'."),
                equalTo(404));
    }

    @Test
    public void deleteExistingInstance() {
        getSetup().client().install();
        createInstance("vs1");
        final List<Instance> instances = getSetup().client().listInstances();
        assertThat(instances, hasSize(1));
        getSetup().client().deleteInstance(instances.get(0).getId());

        assertAll(() -> assertThat(getSetup().client().listInstances(), is(empty())),
                () -> getSetup().exasolMetadata().assertNoConnections(),
                () -> getSetup().exasolMetadata().assertNoVirtualSchema());
    }

    private void createInstance(final String virtualSchemaName) {
        createInstance(config.getExtensionId(), config.getCurrentVersion(), virtualSchemaName);
    }

    private void createInstance(final String extensionId, final String extensionVersion,
            final String virtualSchemaName) {
        getSetup().addVirtualSchemaToCleanupQueue(virtualSchemaName);
        getSetup().addConnectionToCleanupQueue(virtualSchemaName.toUpperCase() + "_CONNECTION");
        final String instanceName = getSetup().client().createInstance(extensionId, extensionVersion,
                createValidParameters(virtualSchemaName));
        assertThat(instanceName, equalTo(virtualSchemaName));
    }

    private List<ParameterValue> createValidParameters(final String virtualSchemaName) {
        final List<ParameterValue> parameters = new ArrayList<>();
        parameters.add(param(VIRTUAL_SCHEMA_NAME_PARAM_NAME, virtualSchemaName));
        parameters.addAll(createValidParameterValues());
        return parameters;
    }

    protected ParameterValue param(final String name, final String value) {
        return new ParameterValue().name(name).value(value);
    }
}
