import countBy from 'lodash/countBy';
import uniq from 'lodash/uniq';

import {ICheckResult, TAssertionResult, TRawAssertionResult, TStructuredAssertion} from 'types/Assertion.types';
import {durationRegExp} from 'constants/Common.constants';
import {Attributes} from 'constants/SpanAttribute.constants';
import {CompareOperatorSymbolMap, OperatorRegexp} from 'constants/Operator.constants';
import {TCompareOperatorSymbol} from '../types/Operator.types';
import {isJson} from '../utils/Common';

const isNumeric = (num: string): boolean => /^-?\d+(?:\.\d+)?$/.test(num);
const isNumericTime = (num: string): boolean => durationRegExp.test(num);

const AssertionService = () => ({
  extractExpectedString(input?: string): string | undefined {
    if (!input) return input;
    const formatted = input.trim();

    if (isJson(input)) return `'${input}'`;

    if (Object.values(Attributes).includes(formatted)) return formatted;
    if (Object.values(Attributes).some(aa => formatted.includes(aa))) return formatted;
    if (isNumeric(formatted) || isNumericTime(formatted)) return formatted;

    const isQuoted = formatted.startsWith('"') && formatted.endsWith('"');

    return isQuoted ? formatted : this.quotedString(formatted);
  },
  quotedString(str: string): string {
    return `\"${str}\"`;
  },
  getSpanIds(resultList: TRawAssertionResult[]) {
    const spanIds = resultList
      .flatMap(assertion => assertion?.spanResults?.map(span => span.spanId ?? '') ?? [])
      .filter(spanId => Boolean(spanId));
    return uniq(spanIds);
  },

  getTotalPassedChecks(resultList: TAssertionResult[]) {
    const passedResults = resultList.flatMap(({spanResults}) => spanResults.map(({passed}) => passed));
    return countBy(passedResults);
  },

  getResultsHashedBySpanId(resultList: TAssertionResult[]) {
    return resultList
      .flatMap(({assertion, spanResults}) => spanResults.map(spanResult => ({result: spanResult, assertion})))
      .reduce((prev: Record<string, ICheckResult[]>, curr) => {
        const items = prev[curr.result.spanId] || [];
        items.push(curr);

        return {
          ...prev,
          [curr.result.spanId]: items,
        };
      }, {});
  },

  getStructuredAssertion(assertion: string): TStructuredAssertion {
    const [left, right] = assertion.split(OperatorRegexp);
    const [comparator = CompareOperatorSymbolMap.EQUALS] = assertion.match(OperatorRegexp) ?? [];

    return {
      left,
      comparator: comparator as TCompareOperatorSymbol,
      right,
    };
  },

  getStringAssertion({left, comparator, right}: TStructuredAssertion): string {
    return `${left} ${comparator} ${right}`;
  },
});

export default AssertionService();
