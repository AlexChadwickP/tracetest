import {SettingOutlined, ToolOutlined} from '@ant-design/icons';

import * as SSpanNode from 'components/Visualization/components/DAG/SpanNode.styled';
import {SemanticGroupNamesToText} from 'constants/SemanticGroupNames.constants';
import {SpanKindToText} from 'constants/Span.constants';
import SpanService from 'services/Span.service';
import {TSpan} from 'types/Span.types';
import * as S from './TestSpecDetail.styled';

interface IProps {
  onSelectSpan(spanId: string): void;
  span?: TSpan;
}

const SpanHeader = ({onSelectSpan, span}: IProps) => {
  const {kind, name, service, system, type} = SpanService.getSpanInfo(span);

  return (
    <S.SpanHeaderContainer onClick={() => onSelectSpan(span?.id ?? '')}>
      <SSpanNode.BadgeType count={SemanticGroupNamesToText[type]} $type={type} />
      <S.HeaderTitle level={3}>{name}</S.HeaderTitle>
      <S.HeaderItem>
        <SettingOutlined />
        <S.HeaderItemText>{`${service} ${SpanKindToText[kind]}`}</S.HeaderItemText>
      </S.HeaderItem>
      {Boolean(system) && (
        <S.HeaderItem>
          <ToolOutlined />
          <S.HeaderItemText>{system}</S.HeaderItemText>
        </S.HeaderItem>
      )}
    </S.SpanHeaderContainer>
  );
};

export default SpanHeader;
