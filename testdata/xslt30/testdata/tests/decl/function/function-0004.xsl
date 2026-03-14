<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://local/functions" exclude-result-prefixes="xs f">
 <xsl:output method="xml" encoding="UTF-8" indent="no" omit-xml-declaration="yes" />
 
 <!-- basic test of first-class functions with a closure using XSLT local variables -->
 
 <xsl:template name="main">
    <x>
      <a><xsl:value-of select="f:round2(2)(1.23456)"/></a>
      <b><xsl:value-of select="function-name(f:round2(2))"/></b>
      <c><xsl:value-of select="function-arity(f:round2(2))"/></c>
      <d><xsl:value-of select="count(f:round2(2))"/></d>         
    </x>
 </xsl:template>
 
 <xsl:function name="f:round2" as="function(*)">
    <xsl:param name="precision" as="xs:integer"/>
    <xsl:sequence select="function($v as xs:decimal) {round($v, $precision)} "/>
 </xsl:function>
</xsl:stylesheet>
