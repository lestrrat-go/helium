<xsl:stylesheet
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:xs="http://www.w3.org/2001/XMLSchema"
 exclude-result-prefixes="xs"
 version="3.0">

<xsl:param name="input-xml" as="xs:string" select="'si-apply-templates-003.xml'"/>

<xsl:template name="main">
<out>
 <xsl:source-document streamable="yes" href="{$input-xml}">
   <xsl:apply-templates select="Report/Document/copy-of()"/>
 </xsl:source-document>
</out> 
</xsl:template>

<xsl:template match="Document">
 <p>Document type: <xsl:value-of select="documentType"/></p>
 <p>Name: <xsl:value-of select="person/(firstname || ' ' || lastname)"/></p>
</xsl:template>

</xsl:stylesheet>