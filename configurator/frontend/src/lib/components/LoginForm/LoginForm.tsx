/* eslint-disable */
import * as React from 'react';
import { Button, Card, Col, Form, Input, message, Modal, Row } from 'antd';

import LockOutlined from '@ant-design/icons/lib/icons/LockOutlined';
import UserOutlined from '@ant-design/icons/lib/icons/UserOutlined';
import MailOutlined from '@ant-design/icons/lib/icons/MailOutlined';

import logo from '../../../icons/logo.svg';
import googleLogo from '../../../icons/google.svg';
import githubLogo from '../../../icons/github.svg';
import './LoginForm.less';
import ApplicationServices from '../../services/ApplicationServices';
import { useState } from 'react';
import { NavLink } from 'react-router-dom';
import { reloadPage } from '../../commons/utils';
import { getErrorPayload } from '@service/analytics';
import { FloatingLabelInput } from '@component/FloatingLabelInput/FloatingLabelInput';

type State = {
  loading: boolean;
  showPasswordReset?: boolean;
};

type Props = {
  errorMessage?: string;
};

export default class LoginForm extends React.Component<Props, State> {
  private services: ApplicationServices;

  constructor(props: Readonly<any>) {
    super(props);
    this.services = ApplicationServices.get();
    this.state = {
      loading: false,
      showPasswordReset: false
    };
  }

  render() {
    let title = (
      <div className="login-form-header-container">
        <img src={logo} alt="[logo]" className="login-form-logo" />
        <span className="login-form-title">Welcome Back!</span>
      </div>
    );
    return (
      <>
        <PasswordResetForm
          key="password-reset-form"
          visible={this.state.showPasswordReset}
          close={() => this.setState({ showPasswordReset: false })}
          onSuccess={() => message.info('Password reset e-mail has been sent!')}
        />
        <>{this.loginFormPart({ title })}</>
      </>
    );
  }

  private async passwordLogin(values) {
    this.setState({ loading: true });
    try {
      await this.services.userService.login(values['username'].trim(), values['password'].trim())
      await this.services.analyticsService.track('app_login', {user: {email: values['username'].trim()}, login_type: 'password'});
      reloadPage();
    } catch (error) {
      message.destroy();
      this.services.analyticsService.track('failed_app_login', {user: {email: values['username'].trim()}, login_type: 'password', ...getErrorPayload(error)});
      console.log('Error', error);
      message.error('Invalid login or password');
    } finally {
      this.setState({ loading: false });
    }
  }

  private async googleLogin() {
    try {
      const email = await this.services.userService.initiateGoogleLogin();
      await this.services.analyticsService.track('app_login', {user: { email }, login_type: 'google'});
      reloadPage();
    } catch (error) {
      message.destroy();
      this.services.analyticsService.track('failed_app_login', {user: {email: null}, login_type: 'google', ...getErrorPayload(error)});
      console.log('Google auth error', error);
      message.error('Access denied: ' + error.message);
    } finally {
      this.setState({ loading: false });
    }
  }

  private async githubLogin() {
    try {
      const email = await this.services.userService.initiateGithubLogin();
      await this.services.analyticsService.track('app_login', {user: { email }, login_type: 'github'});
      reloadPage();
    } catch(error) {
      message.destroy();
      this.services.analyticsService.track('failed_app_login', {user: {email: null}, login_type: 'github', ...getErrorPayload(error)});
      console.log('Github auth error', error);
      message.error('Access denied: ' + error.message);
    } finally {
      this.setState({ loading: false });
    }
  }

  private loginFormPart({ title }) {
    return (
      <Card title={title} style={{ margin: 'auto', marginTop: '100px', maxWidth: '500px' }} bordered={false}>
        <Row>
          <Col span={this.services.userService.getLoginFeatures().oauth ? 12 : 24} className="login-form-left-panel">
            <Form
              name="normal_login"
              className="login-form"
              initialValues={{
                remember: true
              }}
              onFinish={(values) => this.passwordLogin(values)}
              autoComplete="off"
            >
              <FloatingLabelInput
                formName="normal_login"
                name="username"
                rules={[
                  {
                    required: true,
                    message: 'Please, input your e-mail!'
                  },
                  { type: 'email', message: 'Invalid email format' }
                ]}
                floatingLabelText="E-Mail"
                prefix={<MailOutlined />}
                inputType="email"
                className="login-form-input"
              />

              <FloatingLabelInput
                formName="normal_login"
                name="password"
                rules={[
                  {
                    required: true,
                    message: 'Please input your password!'
                  }
                ]}
                floatingLabelText="Password"
                prefix={<LockOutlined className="site-form-item-icon" />}
                inputType="password"
                className="login-form-input"
              />

              <Form.Item>
                <Button
                  key="pwd-login-button"
                  type="primary"
                  htmlType="submit"
                  className="login-form-button"
                  loading={this.state.loading}
                >
                  {this.state.loading ? '' : 'Log in'}
                </Button>
                <div style={{ display: 'flex', justifyContent: 'center' }}>
                  <a className="login-right-forgot" onClick={() => this.setState({ showPasswordReset: true })}>
                    Forgot password?
                  </a>
                </div>
              </Form.Item>
            </Form>
          </Col>

          {this.services.userService.getLoginFeatures().oauth && (
            <Col span={12} className="login-form-right-panel">
              <Form style={{ float: 'right' }}>
                <Form.Item>
                  <Button
                    className="oauth-provider oauth-provider_google"
                    key="google-login-button"
                    icon={<img src={googleLogo} height={16} alt="" />}
                    onClick={(e) => this.googleLogin()}
                  >
                    Sign in with Google
                  </Button>
                </Form.Item>
                <Form.Item>
                  <Button
                    className="oauth-provider oauth-provider_github"
                    key="github-login-button"
                    icon={<img src={githubLogo} height={16} alt="" />}
                    onClick={() => this.githubLogin()}
                  >
                    Sign in with Github
                  </Button>
                </Form.Item>
              </Form>
            </Col>
          )}
        </Row>
        {this.services.userService.getLoginFeatures().signupEnabled && (
          <div className="login-form-signup">
            <div>Don't have an account?</div>
            <Button shape="round" className="login-form-signup-button">
              <NavLink to="/register">Sign Up!</NavLink>
            </Button>
          </div>
        )}
      </Card>
    );
  }
}

function PasswordResetForm({ visible, onSuccess, close }) {
  let services = ApplicationServices.get();
  const [state, setState] = useState({
    loading: false,
    errorMessage: null
  });
  const [form] = Form.useForm();

  const onSubmit = () => {
    setState({ loading: true, errorMessage: null });
    form
      .validateFields()
      .then((values) => {
        services.userService
          .sendPasswordReset(values['email'])
          .then(() => {
            onSuccess();
            close();
            setState({ loading: false, errorMessage: null });
          })
          .catch((error) => {
            message.error(error.message);
            setState({ loading: false, errorMessage: error.message });
          });
      })
      .catch((error) => {
        message.error(error.message);
        setState({ loading: false, errorMessage: error.message });
      });
  };

  return (
    <Modal
      title="Password reset. Please, enter your email"
      visible={visible}
      closable={true}
      onCancel={close}
      footer={[
        <Button key="close" onClick={close}>
          Cancel
        </Button>,
        <Button key="submit" type="primary" loading={state.loading} onClick={onSubmit}>
          Submit
        </Button>
      ]}
    >
      <Form layout="vertical" form={form} name="password-reset-form" className="password-reset-form">
        <Form.Item
          name="email"
          rules={[
            {
              required: true,
              message: "Email can't be empty!"
            }
          ]}
        >
          <Input prefix={<UserOutlined className="site-form-item-icon" />} placeholder="E-mail" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
