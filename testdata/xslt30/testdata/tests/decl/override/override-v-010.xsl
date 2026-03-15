<xsl:package 
  name="http://www.w3.org/xslt30tests/override-v-010"
  package-version="1.0.1"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  <!-- An optional param can override a mandatory param. (See bug 29242) -->
  
  <xsl:use-package name="http://www.w3.org/xslt30tests/override-base-v-003" package-version="*">
    <xsl:override>
      <xsl:param name="req" as="xs:NCName" required="false"/>
    </xsl:override>
  </xsl:use-package>
  
  <xsl:template name="xsl:initial-template" visibility="public">
    <out><xsl:value-of select="$req"/></out>
  </xsl:template>

 
  
</xsl:package>   