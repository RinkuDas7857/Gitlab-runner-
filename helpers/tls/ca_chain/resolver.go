// Inspired by https://github.com/zakjan/cert-chain-resolver/blob/master/certUtil/chain.go
// which is licensed on a MIT license.
//
// Shout out to Jan Žák (http://zakjan.cz) original author of `certUtil` package and other
// contributors who updated it!

package ca_chain

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/zakjan/cert-chain-resolver/certUtil"
)

type resolver interface {
	Resolve(cert *x509.Certificate) ([]*x509.Certificate, error)
}

func newResolver(logger logrus.FieldLogger) resolver {
	return &chainResolver{
		logger: logger,
	}
}

type chainResolver struct {
	logger logrus.FieldLogger
	verifyOptions x509.VerifyOptions
}

func (d *chainResolver) Resolve(cert *x509.Certificate) ([]*x509.Certificate, error) {
	certs, err := d.resolveChain(cert)
	if err != nil {
		return nil, fmt.Errorf("error while resolving certificates chain: %v", err)
	}

	certs, err = d.lookForRootIfMissing(certs)
	if err != nil {
		return nil, fmt.Errorf("error while looking for a missing root certificate: %v", err)
	}

	return certs, err
}

func (d *chainResolver) resolveChain(cert *x509.Certificate) ([]*x509.Certificate, error) {
	certs := make([]*x509.Certificate, 0)
	certs = append(certs, cert)

	for {
		certificate := certs[len(certs)-1]
		log := prepareCertificateLogger(d.logger, certificate)

		if certificate.IssuingCertificateURL == nil {
			log.Debug("[certificates chain build] Certificate doesn't provide parent URL: exiting the loop")
			break
		}

		newCert, err := d.fetchIssuerCertificate(certificate)
		if err != nil {
			return nil, fmt.Errorf("error while fetching issuer certificate: %v", err)
		}

		certs = append(certs, newCert)

		if isChainRootNode(newCert) {
			log.Debug("[certificates chain build] Fetched issuer certificate is a ROOT certificate so exiting the loop")
			break
		}
	}

	return certs, nil
}

func prepareCertificateLogger(logger logrus.FieldLogger, cert *x509.Certificate) logrus.FieldLogger {
	return logger.
		WithFields(logrus.Fields{
			"subject":       cert.Subject.CommonName,
			"issuer":        cert.Issuer.CommonName,
			"serial":        cert.SerialNumber.String(),
			"issuerCertURL": cert.IssuingCertificateURL,
		})
}

func (d *chainResolver) fetchIssuerCertificate(cert *x509.Certificate) (*x509.Certificate, error) {
	log := prepareCertificateLogger(d.logger, cert)
	log.Debug("[certificates chain build] Requesting issuer certificate")

	parentURL := cert.IssuingCertificateURL[0]

	resp, err := http.Get(parentURL)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.
			WithError(err).
			Warning("[certificates chain build] Requesting issuer certificate: HTTP request error")
		return nil, err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.
			WithError(err).
			Warning("[certificates chain build] Requesting issuer certificate: response body read error")
		return nil, err
	}

	newCert, err := certUtil.DecodeCertificate(data)
	if err != nil {
		log.
			WithError(err).
			Warning("[certificates chain build] Requesting issuer certificate: certificate decoding error")
		return nil, err
	}

	log.
		WithFields(logrus.Fields{
			"newCert-subject":       newCert.Subject.CommonName,
			"newCert-issuer":        newCert.Issuer.CommonName,
			"newCert-serial":        newCert.SerialNumber.String(),
			"newCert-issuerCertURL": newCert.IssuingCertificateURL,
		}).
		Debug("[certificates chain build] Requesting issuer certificate: appending the certificate to the chain")

	return newCert, nil
}

func (d *chainResolver) lookForRootIfMissing(certs []*x509.Certificate) ([]*x509.Certificate, error) {
	if len(certs) < 1 {
		return certs, nil
	}

	lastCert := certs[len(certs)-1]

	if isChainRootNode(lastCert) {
		return certs, nil
	}

	prepareCertificateLogger(d.logger, lastCert).
		Debug("[certificates chain build] Verifying last certificate to find the final root certificate")

	verifyChains, err := lastCert.Verify(d.verifyOptions)
	if err != nil {
		if _, e := err.(x509.UnknownAuthorityError); e {
			prepareCertificateLogger(d.logger, lastCert).
				WithError(err).
				Warning("[certificates chain build] Last certificate signed by unknown authority; will not update the chain")

			return certs, nil
		}

		return nil, fmt.Errorf("error while verifying last certificate from the chain: %v", err)
	}

	for _, cert := range verifyChains[0] {
		if lastCert.Equal(cert) {
			continue
		}

		prepareCertificateLogger(d.logger, cert).
			Debug("[certificates chain build] Adding cert from verify chain to the final chain")

		certs = append(certs, cert)
	}

	return certs, nil
}

func isChainRootNode(cert *x509.Certificate) bool {
	return isSelfSigned(cert)
}

func isSelfSigned(cert *x509.Certificate) bool {
	return cert.CheckSignatureFrom(cert) == nil
}
