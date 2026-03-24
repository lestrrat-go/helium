<!-- xsl:expose - fail to access components not exposed as public -->

<xsl:package
  name="http://www.w3.org/xslt30tests/expose-003"  
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/expose-A"
  xmlns:q="http://www.w3.org/xslt30tests/expose-A-private"
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
      <f3><xsl:value-of select="q:f2()"/></f3>
    </out>
  </xsl:template>  
  

</xsl:package>   