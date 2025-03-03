describe('Outputs', () => {
  it('should create outputs from attributes/scratch', () => {
    cy.createTest();

    cy.intercept({method: 'GET', url: '/api/tests/**/run/**/select?query=*'}).as('getSelect');

    // Open output flow from the Trace view (attributes)
    cy.selectRunDetailMode(2);
    cy.get('[data-cy=trace-node-database]', {timeout: 25000}).first().click({force: true});
    cy.get('[data-cy=attributes-search-container] input').type('db.name');
    cy.get('[data-cy=attribute-row-db-name] .ant-dropdown-trigger').click();
    cy.contains('Create output').click();

    // Save output
    cy.wait('@getSelect');
    cy.get('[data-cy=output-save-button]').click();
    cy.get('[data-cy=output-pending-tag]').should('have.length', 1);

    // Add new output from scratch
    cy.get('[data-cy=output-add-button]').click();
    cy.get('form#testOutput').within(() => {
      cy.get('#testOutput_name').type('status_code');
      cy.get('[data-cy=selector-editor] [contenteditable=true]').type('span[tracetest.span.type = "http"]:first');
      cy.get('[data-cy=expression-editor] [contenteditable=true]').type('attr:http.status_code');
    });
    cy.wait('@getSelect');
    cy.get('[data-cy=output-save-button]').click();
    cy.get('[data-cy=output-pending-tag]').should('have.length', 2);

    // Publish and run
    cy.get('[data-cy=output-publish-button]').click();
    cy.wait('@testRuns', {timeout: 30000});
    cy.get('[data-cy=output-item-container]').should('have.length', 2);

    cy.deleteTest(true);
  });

  it('should delete an output', () => {
    cy.createTest();

    cy.intercept({method: 'GET', url: '/api/tests/**/run/**/select?query=*'}).as('getSelect');

    // Open output flow from the Trace view (attributes)
    cy.selectRunDetailMode(2);
    cy.get('[data-cy=trace-node-database]', {timeout: 25000}).first().click({force: true});
    cy.get('[data-cy=attributes-search-container] input').type('db.name');
    cy.get('[data-cy=attribute-row-db-name] .ant-dropdown-trigger').click();
    cy.contains('Create output').click();

    // Save output
    cy.wait('@getSelect');
    cy.get('[data-cy=output-save-button]').click();
    cy.get('[data-cy=output-pending-tag]').should('have.length', 1);

    // Add new output from scratch
    cy.get('[data-cy=output-add-button]').click();
    cy.get('form#testOutput').within(() => {
      cy.get('#testOutput_name').type('status_code');
      cy.get('[data-cy=selector-editor] [contenteditable=true]').type('span[tracetest.span.type = "http"]:first');
      cy.get('[data-cy=expression-editor] [contenteditable=true]').type('attr:http.status_code');
    });
    cy.wait('@getSelect');
    cy.get('[data-cy=output-save-button]').click();
    cy.get('[data-cy=output-pending-tag]').should('have.length', 2);

    // Delete output
    cy.get('[data-cy="output-actions-button-db.name"]').click();
    cy.get('[data-cy=output-item-actions-delete]').click();
    cy.get('[data-cy=delete-confirmation-modal] .ant-btn-primary').click();

    // Publish and run
    cy.get('[data-cy=output-publish-button]').click();
    cy.wait('@testRuns', {timeout: 30000});
    cy.get('[data-cy=output-item-container]').should('have.length', 1);

    cy.deleteTest(true);
  });

  it('should create an output and revert it', () => {
    cy.createTest();

    cy.intercept({method: 'GET', url: '/api/tests/**/run/**/select?query=*'}).as('getSelect');

    // Open output flow from the Trace view (attributes)
    cy.selectRunDetailMode(2);
    cy.get('[data-cy=trace-node-database]', {timeout: 25000}).first().click({force: true});
    cy.get('[data-cy=attributes-search-container] input').type('db.name');
    cy.get('[data-cy=attribute-row-db-name] .ant-dropdown-trigger').click();
    cy.contains('Create output').click();

    // Save output
    cy.wait('@getSelect');
    cy.get('[data-cy=output-save-button]').click();
    cy.get('[data-cy=output-pending-tag]').should('have.length', 1);

    // Revert
    cy.get('[data-cy=output-reset-button]').click();
    cy.get('[data-cy=output-item-container]').should('have.length', 0);

    cy.deleteTest(true);
  });
});
