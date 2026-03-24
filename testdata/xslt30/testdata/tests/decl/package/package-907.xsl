<!-- invalid version attribute on xsl:package  -->
<xsl:package
  name="http://www.w3.org/xslt30tests/package-907" 
  package-version="1.0.0"
  version="three"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  
  
    <xsl:template name="main">
      <not-ok/>
    </xsl:template>
    
  
  
</xsl:package>   