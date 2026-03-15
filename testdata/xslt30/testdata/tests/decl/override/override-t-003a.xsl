<xsl:package 
  name="http://www.w3.org/xslt30tests/override-103a"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/override-base-101"
  exclude-result-prefixes="xs p">
  
  <xsl:use-package
    name="http://www.w3.org/xslt30tests/override-103"
    package-version="1.0.0">
  
    <xsl:override>
  
      <xsl:template name="t-abstract" as="xs:string" visibility="public">
        <xsl:param name="in" as="xs:string"/>
        <xsl:param name="count" as="xs:integer"/>
        <xsl:value-of select="substring($in, 1, $count)"/>
      </xsl:template>

    </xsl:override>
    
    
    
    
  </xsl:use-package>
  
  <xsl:template name="xsl:initial-template" visibility="public">
    <outa>
      <xsl:call-template name="t">
        <xsl:with-param name="in" select="'cornucopia'"/>
        <xsl:with-param name="count" select="4"/>
      </xsl:call-template>
    </outa>
  </xsl:template>
  
  
</xsl:package>   