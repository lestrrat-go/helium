<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
				xmlns:xs="http://www.w3.org/2001/XMLSchema"
				xmlns:my="http://www.schemafunc.example.com/ns/various"
				exclude-result-prefixes="xs my" version="2.0">

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
    </e>
  </xsl:for-each>
  
</xsl:template>

</xsl:stylesheet>