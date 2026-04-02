/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useState, useEffect } from 'react';
import { Modal, Tabs, Typography, Spin, Button, Space } from '@douyinfe/semi-ui';
import { IconCopy } from '@douyinfe/semi-icons';
import { API, showError } from '../../../../helpers';
import { Code } from 'lucide-react';

const { Text } = Typography;

const RequestDetailModal = ({
  showRequestDetail,
  setShowRequestDetail,
  requestDetailTarget,
  t,
}) => {
  const [loading, setLoading] = useState(false);
  const [detail, setDetail] = useState(null);
  const [activeTab, setActiveTab] = useState('request');

  useEffect(() => {
    if (!showRequestDetail || !requestDetailTarget?.request_id) {
      setDetail(null);
      setActiveTab('request');
      return;
    }
    fetchDetail(requestDetailTarget.request_id);
  }, [showRequestDetail, requestDetailTarget]);

  const fetchDetail = async (requestId) => {
    setLoading(true);
    try {
      const res = await API.get(
        `/api/log/detail?request_id=${encodeURIComponent(requestId)}`,
      );
      const { success, message, data } = res.data;
      if (success && data) {
        setDetail(data);
      } else {
        showError(message || t('获取请求详情失败'));
        setDetail(null);
      }
    } catch (err) {
      showError(t('获取请求详情失败') + ': ' + err.message);
      setDetail(null);
    }
    setLoading(false);
  };

  const copyContent = (content) => {
    navigator.clipboard
      .writeText(content)
      .then(() => {})
      .catch(() => {
        showError(t('复制失败'));
      });
  };

  const renderJsonBlock = (content, label) => {
    if (!content) {
      return (
        <div style={{ padding: 20, textAlign: 'center' }}>
          <Text type="tertiary">{t('暂无数据')}</Text>
        </div>
      );
    }

    let formatted = content;
    try {
      // Try to format JSON
      const parsed = JSON.parse(content);
      formatted = JSON.stringify(parsed, null, 2);
    } catch {
      // Not JSON, display as-is
    }

    return (
      <div style={{ position: 'relative' }}>
        <div
          style={{
            position: 'absolute',
            top: 8,
            right: 8,
            zIndex: 1,
          }}
        >
          <Button
            theme="borderless"
            icon={<IconCopy size="small" />}
            onClick={() => copyContent(formatted)}
            size="small"
            style={{ color: 'var(--semi-color-text-2)' }}
          >
            {t('复制')}
          </Button>
        </div>
        <pre
          style={{
            margin: 0,
            padding: '36px 12px 12px',
            maxHeight: '500px',
            overflow: 'auto',
            fontSize: '12px',
            lineHeight: '1.5',
            fontFamily: 'var(--semi-font-family-mono)',
            backgroundColor: 'var(--semi-color-fill-0)',
            borderRadius: '6px',
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
          }}
        >
          {formatted}
        </pre>
      </div>
    );
  };

  const renderHeadersBlock = () => {
    if (!detail?.request_headers) {
      return (
        <div style={{ padding: 20, textAlign: 'center' }}>
          <Text type="tertiary">{t('暂无数据')}</Text>
        </div>
      );
    }

    let headers = {};
    try {
      headers = JSON.parse(detail.request_headers);
    } catch {
      return (
        <div style={{ padding: 12 }}>
          <pre
            style={{
              margin: 0,
              fontSize: '12px',
              fontFamily: 'var(--semi-font-family-mono)',
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
            }}
          >
            {detail.request_headers}
          </pre>
        </div>
      );
    }

    return (
      <div style={{ padding: 12 }}>
        {Object.entries(headers).map(([key, value]) => (
          <div
            key={key}
            style={{
              display: 'flex',
              padding: '6px 0',
              borderBottom: '1px solid var(--semi-color-border)',
              fontSize: '13px',
            }}
          >
            <Text
              strong
              style={{
                width: 180,
                flexShrink: 0,
                fontFamily: 'var(--semi-font-family-mono)',
                fontSize: '12px',
              }}
            >
              {key}
            </Text>
            <Text
              style={{
                flex: 1,
                fontFamily: 'var(--semi-font-family-mono)',
                fontSize: '12px',
                wordBreak: 'break-all',
              }}
            >
              {value}
            </Text>
          </div>
        ))}
      </div>
    );
  };

  const renderMetaInfo = () => {
    if (!detail) return null;
    return (
      <div
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: '16px',
          padding: '12px 16px',
          borderBottom: '1px solid var(--semi-color-border)',
          fontSize: '13px',
        }}
      >
        {detail.request_method && (
          <div>
            <Text type="tertiary" size="small">
              {t('方法')}:{' '}
            </Text>
            <Text strong size="small">
              {detail.request_method}
            </Text>
          </div>
        )}
        {detail.request_path && (
          <div>
            <Text type="tertiary" size="small">
              {t('路径')}:{' '}
            </Text>
            <Text size="small" style={{ fontFamily: 'var(--semi-font-family-mono)', fontSize: '12px' }}>
              {detail.request_path}
            </Text>
          </div>
        )}
        {detail.status_code > 0 && (
          <div>
            <Text type="tertiary" size="small">
              {t('状态码')}:{' '}
            </Text>
            <Text
              strong
              size="small"
              style={{
                color:
                  detail.status_code >= 200 && detail.status_code < 300
                    ? 'var(--semi-color-success)'
                    : detail.status_code >= 400
                      ? 'var(--semi-color-danger)'
                      : 'var(--semi-color-warning)',
              }}
            >
              {detail.status_code}
            </Text>
          </div>
        )}
        {detail.model_name && (
          <div>
            <Text type="tertiary" size="small">
              {t('模型')}:{' '}
            </Text>
            <Text strong size="small">
              {detail.model_name}
            </Text>
          </div>
        )}
        {detail.request_id && (
          <div>
            <Text type="tertiary" size="small">
              Request ID:{' '}
            </Text>
            <Text
              size="small"
              style={{ fontFamily: 'var(--semi-font-family-mono)', fontSize: '12px' }}
            >
              {detail.request_id}
            </Text>
          </div>
        )}
      </div>
    );
  };

  return (
    <Modal
      title={
        <Space>
          <Code size={18} />
          {t('请求详情')}
        </Space>
      }
      visible={showRequestDetail}
      onCancel={() => setShowRequestDetail(false)}
      footer={null}
      centered
      closable
      maskClosable
      width={900}
      bodyStyle={{ maxHeight: '75vh', overflow: 'auto', padding: 0 }}
    >
      <Spin spinning={loading}>
        {detail && (
          <>
            {renderMetaInfo()}
            <Tabs
              type="line"
              activeKey={activeTab}
              onChange={(key) => setActiveTab(key)}
              style={{ padding: '0 16px' }}
            >
              <Tabs.TabPane
                tab={t('请求体')}
                itemKey="request"
              >
                <div style={{ padding: '12px 0 16px' }}>
                  {renderJsonBlock(detail.request_body, 'request_body')}
                </div>
              </Tabs.TabPane>
              <Tabs.TabPane tab={t('请求头')} itemKey="headers">
                <div style={{ padding: '12px 0 16px' }}>
                  {renderHeadersBlock()}
                </div>
              </Tabs.TabPane>
              <Tabs.TabPane tab={t('响应体')} itemKey="response">
                <div style={{ padding: '12px 0 16px' }}>
                  {renderJsonBlock(detail.response_body, 'response_body')}
                </div>
              </Tabs.TabPane>
            </Tabs>
          </>
        )}
        {!loading && !detail && showRequestDetail && (
          <div style={{ padding: 40, textAlign: 'center' }}>
            <Text type="tertiary">{t('暂无请求详情数据')}</Text>
          </div>
        )}
      </Spin>
    </Modal>
  );
};

export default RequestDetailModal;
