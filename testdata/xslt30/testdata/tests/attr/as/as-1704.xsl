<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
	xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" version="2.0">

	<!-- Purpose: Test with local xsl:variable where @select has an xs:untypedAtomic value or 
  				a typed value obtained at run time from element or attribute nodes,  @as is a 
  				built-in derived atomic type. Verify the variable is of the type specified in @as. 
				Types tested: -xs:nonPositiveInteger, xs:int, xs:long, xs:unsignedByte, xs:NMTOKEN, xs:ID -->




	<xsl:template match="/doc-schemaas">
		<out>
			<xsl:variable name="var1" select="/doc-schemaas/elem-NMTOKEN" as="xs:NMTOKEN"/>
			<xsl:variable name="var2" select="/doc-schemaas/elem-int" as="xs:int"/>
			<xsl:variable name="var3" select="/doc-schemaas/elem-nonPositiveInteger"
				as="xs:nonPositiveInteger"/>
			<xsl:variable name="var4" select="/doc-schemaas/elem-long" as="xs:long"/>
			<xsl:variable name="var5" select="/doc-schemaas/elem-unsignedByte" as="xs:unsignedByte"/>
			<xsl:variable name="var6" select="/doc-schemaas/elem-ID-attr/@attr1" as="xs:ID"/>
			<var1>
				<xsl:value-of select="$var1 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$var1 instance of xs:NMTOKEN"/>
			</var1>
			<var2>
				<xsl:value-of select="$var2 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$var2 instance of xs:int"/>
			</var2>
			<var3>
				<xsl:value-of select="$var3 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$var3 instance of xs:nonPositiveInteger"/>
			</var3>
			<var4>
				<xsl:value-of select="$var4 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$var4 instance of xs:long"/>
			</var4>
			<var5>
				<xsl:value-of select="$var5 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$var5 instance of xs:unsignedByte"/>
			</var5>
			<var6>
				<xsl:value-of select="$var6 instance of xs:untypedAtomic"/>
				<xsl:value-of select="$var6 instance of xs:ID"/>
			</var6>
		</out>
	</xsl:template>


</xsl:stylesheet>
