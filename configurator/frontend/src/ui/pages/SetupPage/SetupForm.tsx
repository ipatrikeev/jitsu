/* eslint-disable */
import * as React from 'react';
import { Button, Card, Checkbox, Col, Form, Grid, Input, message, Row, Switch } from 'antd';
import styles from './SetupForm.module.less';

import LockOutlined from '@ant-design/icons/lib/icons/LockOutlined';
import MailOutlined from '@ant-design/icons/lib/icons/MailOutlined';

import { NavLink } from 'react-router-dom';
import { reloadPage } from '@./lib/commons/utils';
import ApplicationServices from '../../../lib/services/ApplicationServices';
import { Align, handleError } from '@./lib/components/components';
import UserOutlined from '@ant-design/icons/lib/icons/UserOutlined';
import BankOutlined from '@ant-design/icons/lib/icons/BankOutlined';
import { useEffect, useState } from 'react';

import fullLogo from '../../../icons/logo.svg';
import { FloatingLabelInput } from '@component/FloatingLabelInput/FloatingLabelInput';
import classNames from 'classnames';
import { validatePassword } from '@./lib/commons/passwordValidator';

import CloseOutlined from '@ant-design/icons/lib/icons/CloseOutlined';
import CheckOutlined from '@ant-design/icons/lib/icons/CheckOutlined';

type State = {
  loading?: boolean;
  tosAgree: boolean;
  emailOptout: boolean;
  usageOptout: boolean;
};

function TermsSwitch({
  children,
  name,
  className = '',
  onChange
}: {
  children: React.ReactNode;
  name: string;
  className?: string;
  onChange: (val: boolean) => void
}) {
  return (
    <Form.Item name={name} className={classNames('mt-4 mb-0', className)} valuePropName="checked" initialValue={true} >
      <Switch checkedChildren={<CheckOutlined />} unCheckedChildren={<CloseOutlined />}  className="inline" defaultChecked={true} onChange={onChange} />
      <span className={styles.termsSwitch}>{children}</span>
    </Form.Item>
  );
}

export default function SetupForm() {
  const appService = ApplicationServices.get();
  const [step, setStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [emailOptout, setEmailOptout] = useState(false);
  const [usageOptout, setUsageOutout] = useState(false);


  const submit = async (values) => {
    setLoading(true)
    try {
      const nonSensitiveUserData = {
        name: values['name'],
        email: values['email'],
        company: values['company_name'],
        emailOptout, usageOptout
      };
      if (!emailOptout) {
        await appService.analyticsService.track('selfhosted_email_subscribe', {
          user: {
            email: nonSensitiveUserData.email,
            name: nonSensitiveUserData.name
          },
          company: nonSensitiveUserData.company,
          emailOptout, usageOptout
        });
      }
      if (!usageOptout) {
        await appService.analyticsService.onUserKnown({
          email: nonSensitiveUserData.email,
          uid: nonSensitiveUserData.email,
          name: nonSensitiveUserData.name
        });
      }
      await appService.analyticsService.track('selfhosted_signup', { usageOptout, emailOptout });
      await appService.userService.setupUser({...nonSensitiveUserData, password: values['password']});

      reloadPage();
    } catch (error) {
      handleError(error);
    } finally {
      setLoading(false)
    }
  };

  return (
    <>
      <div className="flex w-full justify-center px-6">
        <div className="max-w-7xl bg-bgComponent flex-grow my-12 rounded-2xl p-12">
          {step === 0 && (
            <>
              <div className="text-center">
                <img src={fullLogo} className="h-16" />
              </div>
              <h3 className="text-center mt-12 text-3xl">Welcome to Jitsu!</h3>
              <div className="text-center pt-12">
                <Button
                  size="large"
                  type="primary"
                  className={'text-xl h-16 px-24'}
                  onClick={() => {
                    setStep(1);
                  }}
                >
                  Let's get started!
                </Button>
              </div>
            </>
          )}
          {step === 1 && <div className={classNames(styles.registrationForm, 'flex justify-center')}>
            <div className="">
              <div className="text-center">
                <img src={fullLogo} className="h-16 mb-8" />
              </div>
              <h2>What should we call you?</h2>
              <Form name="setup" className="w-full" onFinish={submit}>
                <FloatingLabelInput
                  formName="setup"
                  name="name"
                  rules={[{ required: true, message: 'Please, input your name' }]}
                  floatingLabelText="Your name"
                  prefix={<UserOutlined />}
                  inputType="text"
                  size="large"
                />
                <FloatingLabelInput
                  formName="setup"
                  name="email"
                  rules={[
                    { required: true, message: 'Please, input your name' },
                    {
                      type: 'email',
                      message: 'Invalid email format'
                    }
                  ]}
                  floatingLabelText="E-Mail"
                  prefix={<MailOutlined />}
                  inputType="email"
                  size="large"
                />
                <FloatingLabelInput
                  formName="setup"
                  name="password"
                  rules={[{ required: true, message: 'Please, set the password' }, { validator: validatePassword }]}
                  floatingLabelText="Create a password"
                  prefix={<LockOutlined />}
                  inputType="password"
                  size="large"
                />
                <FloatingLabelInput
                  formName="setup"
                  name="password_confirm"
                  rules={[
                    { required: true, message: 'Please, set the password' },
                    ({ getFieldValue }) => ({
                      validator(_, value) {
                        if (!value || getFieldValue('password') === value) {
                          return Promise.resolve();
                        }
                        return Promise.reject(new Error('The two passwords that you entered do not match!'));
                      }
                    })
                  ]}
                  floatingLabelText="Confirm your password"
                  prefix={<LockOutlined />}
                  inputType="password"
                  size="large"
                />
                <FloatingLabelInput
                  formName="setup"
                  name="company_name"
                  rules={[{ required: true, message: 'Please, tell us a company name' }]}
                  floatingLabelText="Company Name"
                  prefix={<BankOutlined />}
                  inputType="text"
                  size="large"
                />
                <Button htmlType="submit" type="primary" size="large" loading={loading}>
                  Set up Jitsu
                </Button>
                <TermsSwitch name="product-updates" onChange={(val) => setEmailOptout(!val)}>
                  Send me occasional product updates. You may unsubscribe at any time.
                </TermsSwitch>
                <TermsSwitch name="collect-events" onChange={(val) => setUsageOutout(!val)}>
                  Allow Jitsu to anonymously collect usage events.
                </TermsSwitch>

                <ul className={'text-xs text-secondaryText mt-0 pt-0 ml-0 pl-0 list-none'}>
                  <li>Jitsu never collects anything about your data or connection credentials.</li>
                  <li>All collection is completely anonymous.</li>
                  <li>Collection can be turned off at any point in configuration file.</li>
                </ul>
              </Form>
            </div>
          </div>}
        </div>
      </div>
    </>
  );
}
