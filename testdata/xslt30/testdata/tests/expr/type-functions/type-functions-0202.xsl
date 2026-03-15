<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
				xmlns:xs="http://www.w3.org/2001/XMLSchema"
				xmlns:my="http://www.schemafunc.example.com/ns/various"
				exclude-result-prefixes="xs my" version="3.0">

<!-- Test of fn:data#0 -->


<xsl:import-schema namespace="http://www.schemafunc.example.com/ns/various" schema-location="variousTypesSchemaFunc.xsd" />

<xsl:template match="/">
  <out>
  <xsl:apply-templates />
  </out>
</xsl:template>

<xsl:template match="my:userNode/*">
  <xsl:variable name="myDatai" select="." />
  <xsl:for-each select=".">
    <e>
        <xsl:if test="data() instance of xs:decimal"><xsl:value-of select="$myDatai" /> is xs:decimal </xsl:if>
        <xsl:if test="data() instance of xs:float"><xsl:value-of select="$myDatai" /> is xs:float </xsl:if>
        <xsl:if test="data() instance of xs:integer"><xsl:value-of select="$myDatai" /> is xs:integer </xsl:if>
        <xsl:if test="data() instance of xs:date"><xsl:value-of select="$myDatai" /> is xs:date </xsl:if>
        <xsl:if test="data() instance of xs:string"><xsl:value-of select="$myDatai" /> is xs:string </xsl:if>
        <xsl:if test="data() instance of xs:double"><xsl:value-of select="$myDatai" /> is xs:double </xsl:if>
        <xsl:if test="data() instance of xs:positiveInteger"><xsl:value-of select="$myDatai" /> is xs:positiveInteger </xsl:if>
        <xsl:if test="data() instance of xs:nonPositiveInteger"><xsl:value-of select="$myDatai" /> is xs:nonPositiveInteger </xsl:if>
        <xsl:if test="data() instance of xs:negativeInteger"><xsl:value-of select="$myDatai" /> is xs:negativeInteger </xsl:if>
        <xsl:if test="data() instance of xs:long"><xsl:value-of select="$myDatai" /> is xs:long </xsl:if>
        <xsl:if test="data() instance of xs:short"><xsl:value-of select="$myDatai" /> is xs:short </xsl:if>
        <xsl:if test="data() instance of my:elem-decimal-derived-Type"><xsl:value-of select="$myDatai" /> is xs:decimal-derived </xsl:if>
        <xsl:if test="data() instance of my:elem-double-derived-Type"><xsl:value-of select="$myDatai" /> is xs:double-derived </xsl:if>
        <xsl:if test="data() instance of my:elem-float-derived-Type"><xsl:value-of select="$myDatai" /> is xs:float-derived </xsl:if>
        <xsl:if test="data() instance of my:elem-integer-derived-Type"><xsl:value-of select="$myDatai" /> is xs:integer-derived </xsl:if>
        <xsl:if test="data() instance of my:elem-int-derived-Type"><xsl:value-of select="$myDatai" /> is xs:int-derived </xsl:if>
        <xsl:if test="data() instance of my:elem-nonPositiveInteger-derived-Type"><xsl:value-of select="$myDatai" /> is xs:nonPositiveInteger-derived </xsl:if>
        <xsl:if test="data() instance of my:elem-positiveInteger-derived-Type"><xsl:value-of select="$myDatai" /> is xs:positiveInteger-derived </xsl:if>
        <xsl:if test="data() instance of my:elem-negativeInteger-derived-Type"><xsl:value-of select="$myDatai" /> is xs:negativeInteger-derived </xsl:if>
        <xsl:if test="data() instance of my:elem-long-derived-Type"><xsl:value-of select="$myDatai" /> is xs:long-derived </xsl:if>
        <xsl:if test="data() instance of my:elem-unsignedInt-derived-Type"><xsl:value-of select="$myDatai" /> is xs:unsignedInt-derived </xsl:if>
        <xsl:if test="data() instance of my:elem-short-derived-Type"><xsl:value-of select="$myDatai" /> is xs:short-derived </xsl:if>     
    </e>
  </xsl:for-each>
  
</xsl:template>

</xsl:stylesheet>