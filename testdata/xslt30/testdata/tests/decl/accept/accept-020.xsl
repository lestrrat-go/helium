<!-- xsl:accept - conflict between two used packages in the absence of xsl:accept -->

<xsl:package
  name="http://www.w3.org/xslt30tests/accept-007"  
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:A="http://www.w3.org/xslt30tests/accept-A"
  xmlns:B="http://www.w3.org/xslt30tests/accept-B"
  exclude-result-prefixes="xs A B">
  
  <xsl:use-package
     name="http://www.w3.org/xslt30tests/accept-A"
     package-version="1.0.0">
  </xsl:use-package>  
  
  <xsl:use-package
     name="http://www.w3.org/xslt30tests/accept-B"
     package-version="1.0.0">         
  </xsl:use-package>  
  
  <xsl:template name="main" visibility="public">
    <out>
      <v2><xsl:value-of select="$A:v2"/></v2>
    </out>
  </xsl:template>  
  

</xsl:package>   