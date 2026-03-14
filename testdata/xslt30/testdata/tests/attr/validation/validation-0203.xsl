<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0" 
	xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
	xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:rc="http://www.riskcare.com/schvalid023" 
	exclude-result-prefixes="rc">
	
	<!-- validate attribute on xsl:copy-of is ignored if data is not an element or attribute node -->
	
	<xsl:import-schema namespace="http://www.riskcare.com/schvalid023">
		<xs:schema xmlns="http://www.riskcare.com/schvalid023" xmlns:xs="http://www.w3.org/2001/XMLSchema" 
		           targetNamespace="http://www.riskcare.com/schvalid023" 
		           xmlns:r="http://www.riskcare.com/schvalid023" 
		           elementFormDefault="qualified" attributeFormDefault="qualified" version="2">
			<xs:complexType name="DecimalElementType">
				<xs:sequence>
					<xs:element name="amount" type="xs:decimal"/>
				</xs:sequence>
			</xs:complexType>
			<xs:element name="transaction" type="r:DecimalElementType"/>
		</xs:schema>
	</xsl:import-schema>
	
	<xsl:param name="data" select="xs:date('2010-01-25Z')"/>

	<xsl:variable name="copied_data" as="xs:date">
		<xsl:copy-of select="$data" validation="strict"/>
	</xsl:variable>
	
	<xsl:template match="/">
		<root>
           <xsl:sequence select="$copied_data"/>
		</root>
	</xsl:template>

</xsl:stylesheet>