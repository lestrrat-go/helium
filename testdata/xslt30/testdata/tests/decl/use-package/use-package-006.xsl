<xsl:package 
  name="http://www.w3.org/xslt30tests/use-package-006"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  <xsl:use-package
    name="http://www.w3.org/xslt30tests/use-package-base-003"
    package-version="1.0.0"/>

  
  
  
    <!-- Access private variable defined in used package (fails) -->
   
    <xsl:template name="main" visibility="public">
      <xsl:choose>
        <xsl:when test="$v-private = 'v/private'"><ok/></xsl:when>
        <xsl:otherwise><wrong value="$v-private"/></xsl:otherwise>
      </xsl:choose>
    </xsl:template>
    
  
  
</xsl:package>   