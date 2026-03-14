<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0"
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema"
	xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:ns0="urn:gba:sqg"
	exclude-result-prefixes="xs xsi ns0">
	<xsl:output method="xml" encoding="UTF-8" indent="no" />

	<xsl:param name="doc4" select="'stream-500.xml'"/>
	<xsl:strip-space elements="*"/>

	<xsl:template match="/" name="main">
		<root xmlns="urn:gba:sqg" xmlns:ds="http://www.w3.org/2000/09/xmldsig#"
			xmlns:xenc="http://www.w3.org/2001/04/xmlenc#">
			<body>				
                <xsl:for-each select="$doc3/ns0:root/ns0:body/ns0:data_container/ns0:cases">
                    <cases>
                        <xsl:copy-of select="@*" />
                        <xsl:for-each select="ns0:case">
                            <xsl:variable name="lfd" select="@lfdnr" />
                            <case>
                                <xsl:source-document streamable="true" href="{$doc4}">
                                    <xsl:copy-of select="ns0:root/ns0:body/ns0:case[(@lfdnr) = ($lfd)]/ns0:patient" copy-namespaces="false"/>
                                    <test id="{$lfd}" />
                                </xsl:source-document>
                            </case>
                        </xsl:for-each>
                    </cases>
                </xsl:for-each>				
			</body>

		</root>
	</xsl:template>
	
	<xsl:variable name="doc3">
	<root xmlns="urn:gba:sqg" xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"
	xmlns:ds="http://www.w3.org/2000/09/xmldsig#" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
	container_version="2.0" content_version="1.0">
	<header>
		<document>
			<id V="{45664537-5642-9056-8676-456468327446}" />
			<set_id V="{45664537-5642-9056-8676-456468327446}" />
			<origination_dttm V="2001-12-17T09:30:47Z" />
			<modification_dttm V="2011-07-18T11:43:15Z" />
			<software>
				<vendor address="33602 Bielefeld" email="software@vendor.com"
					fax="0123/456789" function="Bundesauswertestelle" name="Vendor"
					phone="0123/456789" registration="V1234" />
				<name V="BASW" />
				<version V="1.01" />
				<release V="3" />
				<specification V="2013 SR 3" />
			</software>
			<data_flow V="QS-Bundesbezogen" />
			<data_target V="Probedatenpool" />
		</document>
		<provider address="33602 Bielefeld" email="office@ProviderXY.com"
			fax="0147/258963" function="Bundesauswertestelle" name="ProviderXY"
			phone="0147/258963" registration="P365478">
			<protocol>
				<validation_item id="1" V="Spezifikation">
					<status V="ERROR">
						<error>
							<error_type V="PID" />
							<list V="1, 2, 3" />
							<rule_id V="1009005" />
							<error_message V="Patienteninformationen wegen Fehler gelöscht" />
						</error>
					</status>
				</validation_item>
			</protocol>
		</provider>
		<protocol>
			<status_document V="OK" />
		</protocol>
		<encryption>
			<ds:KeyInfo>
				<xenc:EncryptedKey Id="qs_data">
					<xenc:EncryptionMethod
						Algorithm="http://www.w3.org/2001/04/xmlenc#rsa-oaep-mgf1p" />
					<xenc:CipherData>
						<xenc:CipherValue>CEPgD38BvEo4uC9yjYGD0skp2tSh8kTEefe94YMSRsUDkYS4Mz4kBPV0x1EnOn9x0FckQCBgAOOu
							IPrhok0eZ+6e28ljrWIfPF5qU66DTrY5So2z2EGHkZQlek+PwUZeXRC3YTWA6HXiE6z1reYIl7WY
							6QaH6/E8JZCdOl8NNdM8/nVgofv2JykjCSmOiwMZztXc9hExq1G2cakOyY9ss4ISrLepFDoGaEDX
							XAlhRQQln/30pzBFOq6/JKDGcgyHRnPbOwSl/WCmDZWTq3gAqKGHDYh/p12Bohx+8sn7vT74Y8gn
							mefQ/RUH/QHc8FupNaoUNpmUNuW4iyiKBoIvMQ==
						</xenc:CipherValue>
					</xenc:CipherData>
				</xenc:EncryptedKey>
			</ds:KeyInfo>
		</encryption>
	</header>
	<body>
		<data_container>
			<care_provider>
				<xenc:EncryptedData Type="http://www.w3.org/2001/04/xmlenc#Content">
					<xenc:EncryptionMethod
						Algorithm="http://www.w3.org/2001/04/xmlenc#aes128-cbc" />
					<xenc:CipherData>
						<xenc:CipherValue>Q5Gb9TuiLdkagsMfwXIgLCETFj5/OPXFNsk3mlwlM9TTxC38qMmj3H4pMQDUSNtDNQ9mQInT9kEo
							hWPOtigi0cxoJVImq4WRlP8qcB9QeMkP2z8eOWkbmpTierEIPGczPkVxzq9v7crOZVhBBUu3HYsO
							1xG1ebx02ulKG0HB+mzBU2twyt211ASK3tfiuw0gkp9nbT8F1elV5OU3YrgkIw==
						</xenc:CipherValue>
					</xenc:CipherData>
				</xenc:EncryptedData>
			</care_provider>
			<cases module="17/7" pseud_procedure="17/5_17/7">
<case lfdnr="1">
					<case_admin>
						<id V="106842" />
						<version V="1" />
						<action V="create" />
						<module V="17/7" />
						<protocol>
							<status_case V="OK" />
						</protocol>
					</case_admin>
					<qs_data module="17/7" xsi:type="qs_data_17n7_type">
						<xenc:EncryptedData Type="http://www.w3.org/2001/04/xmlenc#Content">
							<xenc:EncryptionMethod
								Algorithm="http://www.w3.org/2001/04/xmlenc#aes128-cbc" />
							<xenc:CipherData>
								<xenc:CipherValue>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
								</xenc:CipherValue>
							</xenc:CipherData>
						</xenc:EncryptedData>
					</qs_data>
				</case>
<case lfdnr="2">
					<case_admin>
						<id V="106842" />
						<version V="1" />
						<action V="create" />
						<module V="17/7" />
						<protocol>
							<status_case V="OK" />
						</protocol>
					</case_admin>
					<qs_data module="17/7" xsi:type="qs_data_17n7_type">
						<xenc:EncryptedData Type="http://www.w3.org/2001/04/xmlenc#Content">
							<xenc:EncryptionMethod
								Algorithm="http://www.w3.org/2001/04/xmlenc#aes128-cbc" />
							<xenc:CipherData>
								<xenc:CipherValue>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
								</xenc:CipherValue>
							</xenc:CipherData>
						</xenc:EncryptedData>
					</qs_data>
				</case>
<statistic>
<sent count="2" />
</statistic>
</cases>
</data_container>
</body>
</root>

	</xsl:variable>


</xsl:stylesheet>

