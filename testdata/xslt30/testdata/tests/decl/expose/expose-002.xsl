<!-- xsl:expose - access components exposed as public -->

<xsl:package
  name="http://www.w3.org/xslt30tests/expose-002"  
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/expose-A"
  exclude-result-prefixes="xs p">
  
  <xsl:use-package
     name="http://www.w3.org/xslt30tests/expose-A"
     package-version="1.0.0"/>
  
  <xsl:template name="main" visibility="public">
    <out>
      <t1><xsl:call-template name="t1"/></t1>
      <t2><xsl:call-template name="p:t2"/></t2>
      <f1><xsl:value-of select="p:f1()"/></f1>
      <f2><xsl:value-of select="p:f2()"/></f2>
    </out>
  </xsl:template>  
  

</xsl:package>   