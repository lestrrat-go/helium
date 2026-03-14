<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
                
<?spec xslt#local-variables?>
  
  <xsl:variable name="data">
    <a xmlns:p="http://p.com/ns"/>
  </xsl:variable>
  
  <xsl:param name="prefix" select="'p'"/>
               
  
  <xsl:template match="/">
  <out>
    <xsl:variable name="uri" select="namespace-uri-for-prefix($prefix, $data/*)" as="xs:string"/>
    <xsl:value-of select="$uri"/>
  </out>
  </xsl:template>

  
</xsl:stylesheet>
