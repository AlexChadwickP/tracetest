import {Col, Form, Input, Row} from 'antd';
import {BodyField} from 'components/CreateTestPlugins/Rest/steps/RequestDetails/BodyField/BodyField';
import {IPostmanValues, TDraftTestForm} from 'types/Test.types';
import RequestDetailsAuthInput from '../../../Rest/steps/RequestDetails/RequestDetailsAuthInput/RequestDetailsAuthInput';
import RequestDetailsHeadersInput from '../../../Rest/steps/RequestDetails/RequestDetailsHeadersInput';
import RequestDetailsUrlInput from '../../../Rest/steps/RequestDetails/RequestDetailsUrlInput';
import {CollectionFileField} from './fields/CollectionFileField';
import {EnvFileField} from './fields/EnvFileField';
import {SelectTestFromCollection} from './fields/SelectTestFromCollection';

interface IProps {
  form: TDraftTestForm<IPostmanValues>;
}

const UploadCollectionForm = ({form}: IProps) => {
  return (
    <>
      <Row gutter={12}>
        <Col span={18}>
          <Form.Item name="requests" hidden>
            <Input type="hidden" />
          </Form.Item>
          <Form.Item name="variables" hidden>
            <Input type="hidden" />
          </Form.Item>
          <CollectionFileField form={form} />
          <EnvFileField form={form} />
          <SelectTestFromCollection form={form} />
        </Col>
      </Row>
      <Row gutter={12} style={{marginTop: 16}}>
        <Col span={18}>
          <RequestDetailsUrlInput />
        </Col>
      </Row>
      <Row gutter={12}>
        <Col span={18}>
          <BodyField body={Form.useWatch('body', form)} setBody={body => form.setFieldsValue({body})} />
        </Col>
      </Row>
      <Row gutter={12}>
        <Col span={18}>
          <RequestDetailsHeadersInput />
        </Col>
      </Row>
      <Row gutter={12}>
        <Col span={18}>
          <RequestDetailsAuthInput form={form} />
        </Col>
      </Row>
    </>
  );
};

export default UploadCollectionForm;
