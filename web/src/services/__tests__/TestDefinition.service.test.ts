import TestDefinitionService from '../TestDefinition.service';

describe('TestDefinitionService', () => {
  describe('toRaw', () => {
    it('should return empty response', () => {
      const testResultCount = TestDefinitionService.toRaw({
        assertions: [],
        isDeleted: false,
        isDraft: false,
        originalSelector: '',
        selector: '',
      });
      expect(testResultCount).toEqual({
        assertions: [],
        selector: {
          query: '',
        },
      });
    });
  });
});
