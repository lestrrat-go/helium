<!-- xsl:expose - fail to override components exposed as final -->

<xsl:package
  name="http://www.w3.org/xslt30tests/expose-003"  
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/expose-B"
  xmlns:q="http://www.w3.org/xslt30tests/expose-B-private"
  exclude-result-prefixes="xs p q">
  
  <xsl:use-package
     name="http://www.w3.org/xslt30tests/expose-B"
     package-version="1.0.0">
    
    <xsl:override>
      <xsl:attribute-set name="a1"/>    
    </xsl:override>
    
  </xsl:use-package>  
  
  <xsl:template name="main" visibility="public">
    <out>
      <a1 xsl:use-attribute-sets="a1"/>
      <a2 xsl:use-attribute-sets="p:a2"/>
      <v1><xsl:value-of select="$v1"/></v1>
      <v2><xsl:value-of select="$p:v2"/></v2>
    </out>
  </xsl:template>  
  

</xsl:package>   