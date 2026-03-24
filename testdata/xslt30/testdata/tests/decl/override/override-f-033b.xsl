<xsl:package 
  name="http://www.w3.org/xslt30tests/override-f-033b"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/override-f-033b"
  exclude-result-prefixes="#all">
  
  
  
    <xsl:function name="p:f" as="xs:integer" visibility="public">
      <xsl:param name="in" as="xs:integer*"/>
      <xsl:sequence select="p:g() + (if (empty($in)) then 0 else head($in) + p:h(tail($in)))"/>
    </xsl:function>
  
    <xsl:function name="p:g" as="xs:integer" visibility="public">
      <xsl:sequence select="61"/>
    </xsl:function>
  
    <xsl:function name="p:h" as="xs:integer" visibility="public">
      <xsl:param name="in" as="xs:integer*"/>
      <xsl:sequence select="p:f($in)"/>
    </xsl:function>  
 
  
</xsl:package>   