<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
	xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" version="2.0">

	<!-- Purpose: Test with non-tunnel xsl:with-param where the value in @select is xs:untypedAtomic value 
  				or a typed value, the same as in @as, obtained at run time from element or attribute nodes, 
  				@as is a built-in derived atomic type. Verify the parameter is of the type specified in @as.
				Types tested: -xs:nonPositiveInteger, xs:int, xs:long, xs:unsignedByte, xs:NMTOKEN, xs:ID-->


	<xsl:template match="/">
		<out>
			<xsl:apply-templates>
				<xsl:with-param name="par1" select="/doc-schemaas/elem-NMTOKEN" as="xs:NMTOKEN"
					tunnel="no"/>
				<xsl:with-param name="par2" select="/doc-schemaas/elem-int" as="xs:int" tunnel="no"/>
				<xsl:with-param name="par3" select="/doc-schemaas/elem-nonPositiveInteger"
					as="xs:nonPositiveInteger" tunnel="no"/>
				<xsl:with-param name="par4" select="/doc-schemaas/elem-long" as="xs:long"
					tunnel="no"/>
				<xsl:with-param name="par5" select="/doc-schemaas/elem-unsignedByte"
					as="xs:unsignedByte" tunnel="no"/>
				<xsl:with-param name="par6" select="/doc-schemaas/elem-ID-attr/@attr1" as="xs:ID"
					tunnel="no"/>
			</xsl:apply-templates>
		</out>
	</xsl:template>

	<xsl:template match="doc-schemaas">
		<xsl:param name="par1" select="'1.0'"/>
		<xsl:param name="par2" select="'1.0'"/>
		<xsl:param name="par3" select="'1.0'"/>
		<xsl:param name="par4" select="'1.0'"/>
		<xsl:param name="par5" select="'1.0'"/>
		<xsl:param name="par6" select="'1.0'"/>
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
	</xsl:template>

</xsl:stylesheet>
