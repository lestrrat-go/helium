<xsl:package 
  name="http://www.w3.org/xslt30tests/override-v-008"
  package-version="1.0.1"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  
  <xsl:use-package name="http://www.w3.org/xslt30tests/override-base-v-003" package-version="*">
    <xsl:override>
      <xsl:param name="var" as="xs:int" required="true"/>
    </xsl:override>
  </xsl:use-package>
  
  <xsl:template name="xsl:initial-template" visibility="public">
    <out><xsl:value-of select="$var[. instance of xs:int]"/></out>
  </xsl:template>

 
  
</xsl:package>   