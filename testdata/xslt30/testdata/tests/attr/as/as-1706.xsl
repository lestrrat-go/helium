<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
	xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" version="2.0">

	<!-- Purpose: Test with global xsl:param without @select, where the sequence constructor contains 
  				xs:untypedAtomic or  a typed value obtained at run time, @as is a built-in derived 
  				atomic type. Verify the parameter is of the type specified in @as.Types tested:
				-xs:nonPositiveInteger, xs:int, xs:long, xs:unsignedByte, xs:NMTOKEN, xs:ID -->


	<xsl:param name="par1" as="xs:NMTOKEN">
		<xsl:value-of select="/doc-schemaas/elem-NMTOKEN"/>
	</xsl:param>
	<xsl:param name="par2" as="xs:int">
		<xsl:value-of select="/doc-schemaas/elem-int"/>
	</xsl:param>
	<xsl:param name="par3" as="xs:nonPositiveInteger">
		<xsl:value-of select="/doc-schemaas/elem-nonPositiveInteger"/>
	</xsl:param>
	<xsl:param name="par4" as="xs:long">
		<xsl:value-of select="/doc-schemaas/elem-long"/>
	</xsl:param>
	<xsl:param name="par5" as="xs:unsignedByte">
		<xsl:value-of select="/doc-schemaas/elem-unsignedByte"/>
	</xsl:param>
	<xsl:param name="par6" as="xs:ID">
		<xsl:value-of select="/doc-schemaas/elem-ID-attr/@attr1"/>
	</xsl:param>


	<xsl:template match="/doc-schemaas">
		<out>
			<par1>
				<xsl:value-of select="$par1 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$par1 instance of xs:NMTOKEN"/>
			</par1>
			<par2>
				<xsl:value-of select="$par2 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$par2 instance of xs:int"/>
			</par2>
			<par3>
				<xsl:value-of select="$par3 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$par3 instance of xs:nonPositiveInteger"/>
			</par3>
			<par4>
				<xsl:value-of select="$par4 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$par4 instance of xs:long"/>
			</par4>
			<par5>
				<xsl:value-of select="$par5 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$par5 instance of xs:unsignedByte"/>
			</par5>
			<par6>
				<xsl:value-of select="$par6 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$par6 instance of xs:ID"/>
			</par6>
		</out>
	</xsl:template>


</xsl:stylesheet>
