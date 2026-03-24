<xsl:stylesheet version="2.0" 
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:f="http://fxsl.sf.net/"
xmlns:test="my:test"
  exclude-result-prefixes="f xs test"
>
  <!-- test type conversion of parameters in an xsl:apply-templates call -->
    <?spec xslt#with-param?>
  
  <xsl:template match="/">
   <xsl:variable name="tree"><e>3</e></xsl:variable>
   <xsl:apply-templates select="document('')/*/test:*[1]">
     <xsl:with-param name="arg1" select="$tree/e"/>
   </xsl:apply-templates>
  </xsl:template>
  
  <test:test/>
  <xsl:template name="And" match="test:*">
    <xsl:param name="arg1" as="xs:double"/>
    <out test="{$arg1 instance of xs:double}"/>
  </xsl:template>

</xsl:stylesheet>
