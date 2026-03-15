<!-- syntax variations of xsl:expose -->

<xsl:package
  name="http://www.w3.org/xslt30tests/expose-008a"  
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <xsl:use-package name="http://www.w3.org/xslt30tests/expose-008b" package-version="1.0.0"/>
  
  <xsl:mode/>
  
  <xsl:variable name="test">
    <xxx/>
  </xsl:variable>
  
  <xsl:template name="main" visibility="public">
    <xsl:apply-templates select="$test/*"/>
  </xsl:template>
  
  <xsl:template match="xxx"><out>YYY</out></xsl:template>

</xsl:package>   