<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
	xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
	<!-- Purpose: Test case in which the operand of the test attribute in an xsl:if instruction is a 
  				singleton value of any numeric type or derived from a numeric type, fn:boolean returns 
  				false if the operand value is NaN or is numerically equal to zero; otherwise it returns 
  				true. -->

	<xslt:variable name="v" select="xs:double(/doc/a div /doc/b)"/>

	<xslt:variable name="v1" select="xs:integer('0')"/>

	<xslt:variable name="v2" select="xs:double('0.0000000000000000000000001')"/>

	<xslt:template match="/">
		<out>
			<xslt:if test="$v">
				<xslt:text>NaN Value</xslt:text>
			</xslt:if>
			<xslt:if test="$v1">
				<xslt:text>v1 value </xslt:text>
			</xslt:if>
			<xslt:if test="$v2">
				<xslt:text>v2 value</xslt:text>
			</xslt:if>
		</out>
	</xslt:template>
</xslt:transform>
