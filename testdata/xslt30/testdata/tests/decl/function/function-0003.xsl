<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://local/functions" exclude-result-prefixes="xs f">
 <xsl:output method="xml" encoding="UTF-8" indent="no" omit-xml-declaration="yes" />
 
 <!-- basic test of first-class functions -->
 
 <xsl:template name="main">
    <x>
      <a><xsl:value-of select="f:absf()(-2)"/></a>
      <b><xsl:value-of select="function-name(f:absf())"/></b>
      <c><xsl:value-of select="function-arity(f:absf())"/></c>
      <d><xsl:value-of select="count(f:absf())"/></d>         
    </x>
 </xsl:template>
 
 <xsl:function name="f:absf" as="function(*)">
    <xsl:sequence select="abs#1"/>
 </xsl:function>
</xsl:stylesheet>
