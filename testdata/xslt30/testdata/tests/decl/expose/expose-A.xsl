<!-- simple use of xsl:expose to make templates and functions public -->

<xsl:package
  name="http://www.w3.org/xslt30tests/expose-A"  
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/expose-A"
  xmlns:q="http://www.w3.org/xslt30tests/expose-A-private"
  exclude-result-prefixes="xs p">
  
  <xsl:expose visibility="private" component="*" names="*"/>
  <xsl:expose visibility="public" component="template" names="*"/>
  <xsl:expose visibility="public" component="function" names="p:*"/>
  
  <xsl:variable name="v1" select="0"/>
  <xsl:variable name="p:v2" select="0"/>
  
  <xsl:template name="t1">0</xsl:template>
  <xsl:template name="p:t2">0</xsl:template>
  
  <xsl:function name="p:f1"><xsl:sequence select="1"/></xsl:function>
  <xsl:function name="p:f2"><xsl:sequence select="2"/></xsl:function>
  <xsl:function name="q:f2"><xsl:sequence select="3"/></xsl:function>
  
  <xsl:attribute-set name="a1">
    <xsl:attribute name="A" select="0"/>
  </xsl:attribute-set>
  
  <xsl:attribute-set name="a1">
    <xsl:attribute name="B" select="0"/>
  </xsl:attribute-set>
  
  <xsl:attribute-set name="p:a2">
    <xsl:attribute name="A" select="0"/>
  </xsl:attribute-set>
  
  <xsl:mode name="m1"/>
  <xsl:mode name="p:m2"/>
  
</xsl:package>   